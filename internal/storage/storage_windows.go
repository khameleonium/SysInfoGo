package storage

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v4/disk"
	"github.com/user/sysinfogo/internal/output"
)

type wmiPartition struct {
	DiskIndex   int
	Name        string
	SizeBytes   uint64
	Type        string
	Description string
}

type wmiDisk struct {
	Index         int
	Model         string
	Serial        string
	MediaType     string
	Size          uint64
	InterfaceType string
	Firmware      string
	BytesPerSec   int
	TotalSectors  uint64
	PNPID         string
	Status        string
}

type wmiStaticCache struct {
	disks           []wmiDisk
	partitions      []wmiPartition
	driveToDisk     map[string]int
	partRefToLetter map[string]string
	timestamp       time.Time
	sync.Mutex
}

var cache wmiStaticCache

func getWMIData(ctx context.Context) ([]wmiDisk, []wmiPartition, map[string]int, map[string]string) {
	cache.Lock()
	defer cache.Unlock()

	if time.Since(cache.timestamp) < 30*time.Second {
		return cache.disks, cache.partitions, cache.driveToDisk, cache.partRefToLetter
	}

	cache.disks = queryWMIDisks(ctx)
	cache.partitions = queryWMIPartitions(ctx)
	cache.driveToDisk, cache.partRefToLetter = buildDriveToDiskMap(ctx)
	cache.timestamp = time.Now()

	return cache.disks, cache.partitions, cache.driveToDisk, cache.partRefToLetter
}

func collect(ctx context.Context) (*Info, []output.Warning, error) {
	var warns []output.Warning

	wmiDisks, wmiPartitions, driveToDisk, partRefToLetter := getWMIData(ctx)

	parts, _ := disk.PartitionsWithContext(ctx, false)

	partitionByDisk := make(map[int][]wmiPartition)
	for _, wp := range wmiPartitions {
		partitionByDisk[wp.DiskIndex] = append(partitionByDisk[wp.DiskIndex], wp)
	}

	usageByLetter := make(map[string]*disk.UsageStat)
	for _, p := range parts {
		letter := strings.TrimRight(strings.TrimRight(p.Mountpoint, `\`), `:`)
		if u, err := disk.UsageWithContext(ctx, p.Mountpoint); err == nil {
			uCopy := *u
			usageByLetter[letter] = &uCopy
		}
	}

	var disks []DiskInfo
	seenDisks := make(map[int]int)

	for _, p := range parts {
		letter := strings.TrimRight(strings.TrimRight(p.Mountpoint, `\`), `:`)
		diskNum, ok := driveToDisk[letter]
		if !ok {
			continue
		}

		idx, ok2 := seenDisks[diskNum]
		if !ok2 {
			phys := findWMIByDiskNumber(wmiDisks, diskNum)
			if phys.Model == "" {
				phys.Model = diskName(diskNum)
			}

			if wmParts, ok := partitionByDisk[diskNum]; ok {
				for _, wp := range wmParts {
					wpLetter, hasLetter := partRefToLetter[partRefKey(wp.Name)]
					if hasLetter && wpLetter != "" {
						continue
					}
					phys.Partitions = append(phys.Partitions, PartitionInfo{
						Name:    wp.Name,
						FSType:  classifyPartType(wp.Type),
						TotalGB: float64(wp.SizeBytes) / (1024 * 1024 * 1024),
						Type:    wpDescription(wp),
						Hidden:  true,
					})
				}
			}

			disks = append(disks, phys)
			idx = len(disks) - 1
			seenDisks[diskNum] = idx
		}

		u := usageByLetter[letter]
		if u == nil {
			continue
		}

		part := PartitionInfo{
			Name:       p.Device,
			MountPoint: strings.TrimRight(p.Mountpoint, `\`),
			FSType:     p.Fstype,
			TotalGB:    float64(u.Total) / (1024 * 1024 * 1024),
			FreeGB:     float64(u.Free) / (1024 * 1024 * 1024),
			UsedPct:    u.UsedPercent,
		}

		if wmParts, ok := partitionByDisk[diskNum]; ok {
			for _, wp := range wmParts {
				wpLetter, hasLetter := partRefToLetter[wp.Name]
				if hasLetter && strings.EqualFold(wpLetter, letter) {
					part.Type = wpDescription(wp)
					break
				}
			}
		}

		disks[idx].Partitions = append(disks[idx].Partitions, part)
		disks[idx].SizeGB = max(disks[idx].SizeGB, float64(wmiDiskSizeByNum(wmiDisks, diskNum))/(1024*1024*1024))
		if disks[idx].Interface == "" {
			disks[idx].Interface = detectInterface(disks[idx].Interface, disks[idx].Model, int(part.TotalGB))
		}
	}

	for _, wd := range wmiDisks {
		if _, seen := seenDisks[wd.Index]; seen {
			continue
		}
		d := wmiDiskToInfo(wd)
		if wmParts, ok := partitionByDisk[wd.Index]; ok {
			for _, wp := range wmParts {
				wpLetter, hasLetter := partRefToLetter[wp.Name]
				if hasLetter && wpLetter != "" {
					continue
				}
				d.Partitions = append(d.Partitions, PartitionInfo{
					Name:    wp.Name,
					FSType:  classifyPartType(wp.Type),
					TotalGB: float64(wp.SizeBytes) / (1024 * 1024 * 1024),
					Type:    wpDescription(wp),
					Hidden:  true,
				})
			}
		}
		disks = append(disks, d)
	}

	ramDisks := detectRAMDisks(ctx, usageByLetter, driveToDisk)
	for _, rd := range ramDisks {
		rde := DiskInfo{
			Model:     fmt.Sprintf("RAM-диск (%s:)", rd.Name),
			Interface: "RAM Disk",
			MediaType: "RAM",
			IsRAMDisk: true,
			SizeGB:    float64(rd.Total) / (1024 * 1024 * 1024),
			Health:    "OK",
			HealthPct: 100,
			Partitions: []PartitionInfo{{
				Name:       fmt.Sprintf("%s:", rd.Name),
				MountPoint: fmt.Sprintf("%s:", rd.Name),
				FSType:     rd.FSType,
				TotalGB:    float64(rd.Total) / (1024 * 1024 * 1024),
				FreeGB:     float64(rd.Free) / (1024 * 1024 * 1024),
				UsedPct:    float64(rd.Total-rd.Free) / float64(rd.Total) * 100,
			}},
		}
		disks = append(disks, rde)
	}

	hasSmartctl := false
	if smartctl, _ := exec.LookPath("smartctl"); smartctl != "" {
		hasSmartctl = true
	}

	warnedSmartctl := false
	for i := range disks {
		if !disks[i].IsRAMDisk {
			if hasSmartctl {
				w := collectSmartWindows(ctx, &disks[i])
				warns = append(warns, w...)
			} else if !warnedSmartctl {
				warns = append(warns, output.Warning{
					Section: "storage",
					Message: "Утилита smartctl не найдена.",
					OSHint:  "Пожалуйста, установите пакет smartmontools, чтобы видеть данные о здоровье дисков (SMART).",
				})
				warnedSmartctl = true
			}
		}
	}

	return &Info{Disks: disks}, warns, nil
}

type ramDisk struct {
	Name, FSType string
	Total, Free  uint64
}

func detectRAMDisks(ctx context.Context, usageByLetter map[string]*disk.UsageStat, driveToDisk map[string]int) []ramDisk {
	var result []ramDisk

	letters := make(map[string]bool)
	for letter := range usageByLetter {
		letters[letter] = true
	}

	raw := queryWMIList(ctx, "logicaldisk", "get", "DeviceID,Description,FileSystem,Size,FreeSpace,VolumeName")
	if raw == "" {
		return result
	}

	type ldInfo struct {
		DeviceID    string
		Description string
		FileSystem  string
		Size        uint64
		FreeSpace   uint64
		VolumeName  string
	}

	var entries []ldInfo
	var cur *ldInfo
	sc := bufio.NewScanner(strings.NewReader(raw))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			if cur != nil {
				entries = append(entries, *cur)
				cur = nil
			}
			continue
		}
		if cur == nil {
			cur = &ldInfo{}
		}
		if v := extractTag(line, "DeviceID="); v != "" {
			cur.DeviceID = v
		}
		if v := extractTag(line, "Description="); v != "" {
			cur.Description = v
		}
		if v := extractTag(line, "FileSystem="); v != "" {
			cur.FileSystem = v
		}
		if v := extractTag(line, "Size="); v != "" {
			fmt.Sscanf(v, "%d", &cur.Size)
		}
		if v := extractTag(line, "FreeSpace="); v != "" {
			fmt.Sscanf(v, "%d", &cur.FreeSpace)
		}
		if v := extractTag(line, "VolumeName="); v != "" {
			cur.VolumeName = v
		}
	}
	if cur != nil {
		entries = append(entries, *cur)
	}

	for _, e := range entries {
		letter := strings.TrimSuffix(strings.TrimSpace(e.DeviceID), ":")
		upper := strings.ToUpper(e.Description + " " + e.VolumeName)
		isRAM := strings.Contains(upper, "RAM")

		if isRAM {
			if _, inMap := driveToDisk[letter]; inMap {
				continue
			}
			result = append(result, ramDisk{
				Name:   letter,
				FSType: e.FileSystem,
				Total:  e.Size,
				Free:   e.FreeSpace,
			})
			delete(usageByLetter, letter)
		}
	}

	return result
}


func wmiDiskSizeByNum(disks []wmiDisk, num int) uint64 {
	for _, wd := range disks {
		if wd.Index == num {
			return wd.Size
		}
	}
	return 0
}

func wmiDiskToInfo(wd wmiDisk) DiskInfo {
	d := DiskInfo{
		DiskNumber:       wd.Index,
		DeviceName:       fmt.Sprintf("pd%d", wd.Index),
		Model:            wd.Model,
		Serial:           wd.Serial,
		MediaType:        classifyMedia(wd.MediaType, wd.InterfaceType, wd.Model),
		FirmwareRevision: wd.Firmware,
		BytesPerSector:   wd.BytesPerSec,
		TotalSectors:     wd.TotalSectors,
		PNPID:            wd.PNPID,
	}
	if wd.Status != "" {
		d.Health = wd.Status
		if wd.Status == "OK" {
			d.HealthPct = 100
		} else {
			d.HealthPct = 10
		}
	} else {
		d.Health = "Unknown"
		d.HealthPct = 0
	}
	if wd.Size > 0 {
		d.SizeGB = float64(wd.Size) / (1024 * 1024 * 1024)
	}
	if iface := classifyInterface(wd.InterfaceType, wd.PNPID, wd.Model); iface != "" {
		d.Interface = iface
	}
	return d
}

func classifyPartType(partType string) string {
	upper := strings.ToUpper(partType)
	if strings.Contains(upper, "SYSTEM") || strings.Contains(upper, "EFI") {
		return "FAT32"
	}
	if strings.Contains(upper, "RECOVERY") {
		return "NTFS"
	}
	return ""
}

func queryWMIPartitions(ctx context.Context) []wmiPartition {
	raw := queryWMIList(ctx, "partition", "get", "DiskIndex,Name,Size,Type,Description")
	if raw == "" {
		return nil
	}

	var result []wmiPartition
	var cur *wmiPartition
	sc := bufio.NewScanner(strings.NewReader(raw))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			if cur != nil {
				result = append(result, *cur)
				cur = nil
			}
			continue
		}
		if cur == nil {
			cur = &wmiPartition{DiskIndex: -1}
		}
		if v := extractTag(line, "DiskIndex="); v != "" {
			fmt.Sscanf(v, "%d", &cur.DiskIndex)
		}
		if v := extractTag(line, "Name="); v != "" {
			cur.Name = v
		}
		if v := extractTag(line, "Size="); v != "" {
			fmt.Sscanf(v, "%d", &cur.SizeBytes)
		}
		if v := extractTag(line, "Type="); v != "" {
			cur.Type = v
		}
		if v := extractTag(line, "Description="); v != "" {
			cur.Description = v
		}
	}
	if cur != nil {
		result = append(result, *cur)
	}

	return result
}

func buildDriveToDiskMap(ctx context.Context) (map[string]int, map[string]string) {
	driveToDisk := make(map[string]int)
	partRefToLetter := make(map[string]string)

	diskIdxByPart := queryDiskToPartitionMap(ctx)
	letterByPart := queryPartitionToLetterMap(ctx)

	for partRef, diskIdx := range diskIdxByPart {
		if letter, ok := letterByPart[partRef]; ok {
			driveToDisk[letter] = diskIdx
			partRefToLetter[partRef] = letter
		}
	}

	return driveToDisk, partRefToLetter
}

func wpDescription(wp wmiPartition) string {
	if wp.Description != "" {
		return wp.Description
	}
	if wp.Type != "" {
		return wp.Type
	}
	return ""
}

func partRefKey(partName string) string {
	if strings.Contains(partName, "#") {
		partName = strings.Replace(partName, "Disk #", "", 1)
		partName = strings.Replace(partName, "Partition #", "", 1)
		parts := strings.SplitN(partName, ",", 2)
		if len(parts) == 2 {
			return strings.TrimSpace(parts[0]) + "," + strings.TrimSpace(parts[1])
		}
	}
	return partName
}

func queryDiskToPartitionMap(ctx context.Context) map[string]int {
	result := make(map[string]int)
	raw := queryWMIList(ctx, "path", "Win32_DiskDriveToDiskPartition", "get", "Antecedent,Dependent")
	if raw == "" {
		return result
	}

	var curDisk, curPart int
	diskFound, partFound := false, false

	sc := bufio.NewScanner(strings.NewReader(raw))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			if diskFound || partFound {
				diskFound, partFound = false, false
				key := fmt.Sprintf("%d,%d", curDisk, curPart)
				result[key] = curDisk
			}
			continue
		}

		if v := extractTag(line, "Antecedent="); v != "" {
			curDisk = parsePhysicalDrive(v)
			diskFound = true
		}
		if v := extractTag(line, "Dependent="); v != "" {
			if n := parsePartitionNumber(v); n >= 0 {
				curPart = n
				partFound = true
			}
		}
	}
	if diskFound || partFound {
		key := fmt.Sprintf("%d,%d", curDisk, curPart)
		result[key] = curDisk
	}

	return result
}

func queryPartitionToLetterMap(ctx context.Context) map[string]string {
	result := make(map[string]string)
	raw := queryWMIList(ctx, "path", "Win32_LogicalDiskToPartition", "get", "Antecedent,Dependent")
	if raw == "" {
		return result
	}

	var curDisk, curPart int
	var curLetter string
	partFound, letterFound := false, false

	sc := bufio.NewScanner(strings.NewReader(raw))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			if partFound && letterFound {
				key := fmt.Sprintf("%d,%d", curDisk, curPart)
				result[key] = curLetter
				partFound, letterFound = false, false
			}
			continue
		}

		if v := extractTag(line, "Antecedent="); v != "" {
			if d, p := parseDiskAndPartition(v); d >= 0 {
				curDisk = d
				curPart = p
				partFound = true
			}
		}
		if v := extractTag(line, "Dependent="); v != "" {
			curLetter = extractDriveLetter(v)
			if curLetter != "" {
				letterFound = true
			}
		}
	}
	if partFound && letterFound {
		key := fmt.Sprintf("%d,%d", curDisk, curPart)
		result[key] = curLetter
	}

	return result
}

func queryWMIList(ctx context.Context, args ...string) string {
	cmd := exec.CommandContext(ctx, "wmic", args...)
	cmd.Args = append(cmd.Args, "/format:list")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return string(out)
}

func extractTag(line, prefix string) string {
	idx := strings.Index(line, prefix)
	if idx < 0 {
		return ""
	}
	return strings.TrimSpace(line[idx+len(prefix):])
}

func parsePhysicalDrive(value string) int {
	idx := strings.Index(value, "PHYSICALDRIVE")
	if idx < 0 {
		return -1
	}
	rest := value[idx+len("PHYSICALDRIVE"):]
	numStr := ""
	for _, ch := range rest {
		if ch >= '0' && ch <= '9' {
			numStr += string(ch)
		} else {
			break
		}
	}
	n, _ := strconv.Atoi(numStr)
	return n
}

func parsePartitionNumber(value string) int {
	idx := strings.Index(value, "Partition #")
	if idx < 0 {
		return -1
	}
	rest := value[idx+len("Partition #"):]
	numStr := ""
	for _, ch := range rest {
		if ch >= '0' && ch <= '9' {
			numStr += string(ch)
		} else {
			break
		}
	}
	n, _ := strconv.Atoi(numStr)
	return n
}

func parseDiskAndPartition(value string) (int, int) {
	diskIdx := strings.Index(value, "Disk #")
	partIdx := strings.Index(value, "Partition #")
	if diskIdx < 0 || partIdx < 0 {
		return -1, -1
	}

	diskStr := ""
	for _, ch := range value[diskIdx+len("Disk #"):] {
		if ch >= '0' && ch <= '9' {
			diskStr += string(ch)
		} else {
			break
		}
	}

	partStr := ""
	for _, ch := range value[partIdx+len("Partition #"):] {
		if ch >= '0' && ch <= '9' {
			partStr += string(ch)
		} else {
			break
		}
	}

	d, _ := strconv.Atoi(diskStr)
	p, _ := strconv.Atoi(partStr)
	return d, p
}

func extractDriveLetter(value string) string {
	idx := strings.Index(value, `DeviceID="`)
	if idx < 0 {
		return ""
	}
	rest := value[idx+len(`DeviceID="`):]
	end := strings.Index(rest, `"`)
	if end < 0 {
		return ""
	}
	val := rest[:end]
	return strings.TrimSuffix(strings.TrimSuffix(val, ":"), ":")
}

func queryWMIDisks(ctx context.Context) []wmiDisk {
	raw := queryWMIList(ctx, "diskdrive", "get",
		"Index,Model,SerialNumber,MediaType,Size,InterfaceType,FirmwareRevision,BytesPerSector,TotalSectors,PNPDeviceID,Status")
	if raw == "" {
		return nil
	}

	return parseWMIListDisks(raw)
}

func parseWMIListDisks(raw string) []wmiDisk {
	var disks []wmiDisk
	var cur *wmiDisk

	sc := bufio.NewScanner(strings.NewReader(raw))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			if cur != nil {
				disks = append(disks, *cur)
				cur = nil
			}
			continue
		}

		if cur == nil {
			cur = &wmiDisk{Index: -1}
		}

		if v := extractTag(line, "Index="); v != "" {
			fmt.Sscanf(v, "%d", &cur.Index)
		}
		if v := extractTag(line, "Model="); v != "" {
			cur.Model = v
		}
		if v := extractTag(line, "SerialNumber="); v != "" {
			cur.Serial = v
		}
		if v := extractTag(line, "MediaType="); v != "" {
			cur.MediaType = v
		}
		if v := extractTag(line, "Size="); v != "" {
			fmt.Sscanf(v, "%d", &cur.Size)
		}
		if v := extractTag(line, "InterfaceType="); v != "" {
			cur.InterfaceType = v
		}
		if v := extractTag(line, "FirmwareRevision="); v != "" {
			cur.Firmware = v
		}
		if v := extractTag(line, "BytesPerSector="); v != "" {
			fmt.Sscanf(v, "%d", &cur.BytesPerSec)
		}
		if v := extractTag(line, "TotalSectors="); v != "" {
			fmt.Sscanf(v, "%d", &cur.TotalSectors)
		}
		if v := extractTag(line, "PNPDeviceID="); v != "" {
			cur.PNPID = v
		}
		if v := extractTag(line, "Status="); v != "" {
			cur.Status = v
		}
	}
	if cur != nil {
		disks = append(disks, *cur)
	}

	return disks
}

func findWMIByDiskNumber(disks []wmiDisk, num int) DiskInfo {
	for _, wd := range disks {
		if wd.Index == num {
			return wmiDiskToInfo(wd)
		}
	}
	return DiskInfo{}
}

func classifyInterface(ifaceType, pnpID, model string) string {
	upper := strings.ToUpper(ifaceType + " " + pnpID + " " + model)
	if strings.Contains(upper, "NVME") {
		return "NVMe"
	}
	if strings.Contains(upper, "USB") {
		return "USB"
	}
	return "SATA"
}

func classifyMedia(mediaType, ifaceType, model string) string {
	upper := strings.ToUpper(mediaType + " " + ifaceType + " " + model)
	if strings.Contains(upper, "SSD") || strings.Contains(upper, "NVME") {
		return "SSD"
	}
	if strings.Contains(upper, "USB") || strings.Contains(upper, "REMOVABLE") {
		return "USB"
	}
	isSSD := strings.Contains(upper, "KINGSTON") ||
		(strings.Contains(upper, "SAMSUNG") && strings.Contains(upper, "EVO")) ||
		strings.Contains(upper, "CRUCIAL") ||
		strings.Contains(upper, "WDS") ||
		(strings.Contains(upper, "SANDISK") && strings.Contains(upper, "SSD"))
	if isSSD {
		return "SSD"
	}
	return "HDD"
}

func detectInterface(iface, model string, sizeGB int) string {
	if iface != "" {
		return iface
	}
	upper := strings.ToUpper(model)
	if strings.Contains(upper, "NVME") {
		return "NVMe"
	}
	return "SATA"
}

func diskName(num int) string {
	return fmt.Sprintf("Disk %d", num)
}

func collectSmartWindows(ctx context.Context, d *DiskInfo) []output.Warning {
	if smartPath, _ := exec.LookPath("smartctl"); smartPath == "" {
		return nil
	}

	target := ""
	if len(d.Partitions) > 0 {
		for _, p := range d.Partitions {
			if !p.Hidden && p.MountPoint != "" {
				target = p.MountPoint
				break
			}
		}
	}
	if target == "" {
		return nil
	}
	if !strings.HasSuffix(target, ":") {
		target += ":"
	}

	smartCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	cmd := exec.CommandContext(smartCtx, "smartctl", "-A", target)
	out, err := cmd.Output()
	if err != nil {
		return nil
	}

	parseSmartOutput(string(out), d)
	return nil
}
