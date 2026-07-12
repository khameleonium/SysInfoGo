 = @(
    @("windows", "amd64", "sysinfogo_windows_amd64.exe"),
    @("windows", "386", "sysinfogo_windows_386.exe"),
    @("linux", "amd64", "sysinfogo_linux_amd64"),
    @("linux", "arm64", "sysinfogo_linux_arm64"),
    @("linux", "386", "sysinfogo_linux_386"),
    @("darwin", "amd64", "sysinfogo_darwin_amd64"),
    @("darwin", "arm64", "sysinfogo_darwin_arm64")
)

foreach ( in ) {
     = [0]
     = [1]
     = [2]
    Write-Host "Building for /..."
     = 
     = 
    go build -ldflags="-s -w" -o "bin/" ./cmd/sysinfogo
}