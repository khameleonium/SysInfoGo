let localeDict = {};
let currentData = null;
let showAllProcesses = false;

window.T = function(key) {
    return localeDict[key] || key;
};

window.toggleProcesses = function() {
    showAllProcesses = !showAllProcesses;
    if (currentData && currentData.processes) {
        renderProcesses(currentData.processes);
    }
};

function translateDOM() {
    document.querySelectorAll('[data-i18n]').forEach(el => {
        el.textContent = T(el.getAttribute('data-i18n'));
    });
}

document.addEventListener('DOMContentLoaded', async () => {
    // Load locale
    try {
        const res = await fetch('/api/locale');
        if (res.ok) {
            localeDict = await res.json();
            translateDOM();
        }
    } catch (e) {
        console.error("Failed to load locale:", e);
    }

    const injectedData = document.getElementById('injected-data').textContent.trim();
    if (injectedData) {
        // HTML Export mode
        try {
            const data = JSON.parse(injectedData);
            renderData(data);
        } catch (e) {
            console.error("Failed to parse injected data:", e);
        }
    } else {
        // Web Server mode
        document.getElementById('live-badge').style.display = 'inline-block';
        document.getElementById('export-buttons').style.display = 'flex';
        fetchData();
        setInterval(fetchData, 2000);
    }
});

async function fetchData() {
    try {
        const res = await fetch('/api/info');
        if (res.ok) {
            const data = await res.json();
            renderData(data);
        }
    } catch (e) {
        console.error("Fetch error:", e);
    }
}

function renderData(data) {
    if (!data) return;
    currentData = data;
    if (data.summary) renderHeader(data);
    if (data.summary) renderSummary(data.summary);
    if (data.cpu || data.memory) {
        document.getElementById('card-cpuram').style.display = 'flex';
        if (data.cpu) renderCPU(data.cpu);
        if (data.memory) renderMemory(data.memory);
    }
    if (data.network && data.network.interfaces) {
        window.netHistoryMap = window.netHistoryMap || {};
        data.network.interfaces.forEach(iface => {
            let val = (iface.speed_recv_mbps || 0) + (iface.speed_sent_mbps || 0);
            if (!window.netHistoryMap[iface.name]) {
                window.netHistoryMap[iface.name] = Array(30).fill(0);
            }
            window.netHistoryMap[iface.name].push(val);
            if (window.netHistoryMap[iface.name].length > 50) {
                window.netHistoryMap[iface.name].shift();
            }
        });
        if (window.activeNetGraph && window.renderNetGraphModal) {
            window.renderNetGraphModal();
        }
    }

    window.tempHistoryMap = window.tempHistoryMap || {};
    if (data.cpu && data.cpu.package_temp > 0) {
        let h = window.tempHistoryMap["cpu"] || Array(20).fill(data.cpu.package_temp);
        h.push(data.cpu.package_temp);
        if (h.length > 50) h.shift();
        window.tempHistoryMap["cpu"] = h;
    }
    if (data.gpu && data.gpu.gpus) {
        data.gpu.gpus.forEach((g) => {
            if (g.temp_c > 0) {
                let key = "gpu_" + g.name;
                let h = window.tempHistoryMap[key] || Array(20).fill(g.temp_c);
                h.push(g.temp_c);
                if (h.length > 50) h.shift();
                window.tempHistoryMap[key] = h;
            }
        });
    }
    if (data.storage && data.storage.disks) {
        data.storage.disks.forEach(d => {
            if (d.temp_c > 0) {
                let key = "disk_" + d.model;
                let h = window.tempHistoryMap[key] || Array(20).fill(d.temp_c);
                h.push(d.temp_c);
                if (h.length > 50) h.shift();
                window.tempHistoryMap[key] = h;
            }
        });
    }
    if (window.activeTempGraph && window.renderTempGraphModal) {
        window.renderTempGraphModal();
    }

    if (data.gpu) renderGPU(data.gpu);
    if (data.battery) renderBattery(data.battery);
    if (data.network) renderNetwork(data.network);
    if (data.processes) renderProcesses(data.processes);
    if (data.storage) renderStorage(data.storage);
}

function formatGB(gb) {
    if (gb < 1) return (gb * 1024).toFixed(1) + ' MB';
    return gb.toFixed(2) + ' GB';
}

function getProgressColorClass(pct) {
    if (pct > 90) return 'bg-danger';
    if (pct > 75) return 'bg-warn';
    return 'bg-accent';
}

function renderHeader(data) {
    const meta = document.getElementById('header-meta');
    meta.innerHTML = `
        <div><span class="os-name">${data.summary.os}</span> (${data.summary.arch})</div>
        <div>Uptime: ${data.summary.uptime}</div>
        <div>${T('Обновлено')}: ${data.timestamp || new Date().toLocaleTimeString()}</div>
    `;
}

function kvPair(key, val, valClass = '') {
    return `<div class="kv-pair"><span class="kv-key">${key}</span><span class="kv-val ${valClass}">${val}</span></div>`;
}

function progressBar(pct, labelLeft, labelRight) {
    const colorClass = getProgressColorClass(pct);
    return `
        <div>
            <div class="kv-pair" style="font-size: 0.8rem;">
                <span>${labelLeft}</span>
                <span>${labelRight} (${pct.toFixed(1)}%)</span>
            </div>
            <div class="progress-container">
                <div class="progress-bar ${colorClass}" style="width: ${pct}%"></div>
            </div>
        </div>
    `;
}

function renderSummary(summary) {
    const el = document.getElementById('content-summary');
    document.getElementById('card-summary').style.display = 'flex';
    let html = kvPair(T('Система'), summary.os);
    if (summary.kernel) html += kvPair(T('Ядро'), summary.kernel);
    if (summary.arch) html += kvPair(T('Архитектура'), summary.arch);
    if (summary.hostname) html += kvPair(T('Хост'), summary.hostname);
    if (summary.uptime) html += kvPair(T('Uptime'), summary.uptime);
    if (summary.boot_time) html += kvPair(T('Запуск'), summary.boot_time);
    if (summary.motherboard) html += kvPair(T('Материнская плата'), summary.motherboard);
    if (summary.virtualization) html += kvPair(T('Виртуализация'), summary.virtualization, 'color-accent');
    if (summary.primary_ip) html += kvPair('IP', summary.primary_ip);
    
    el.innerHTML = html;
}

function makeTempSparklineSVG(data) {
    if (!data || data.length < 2) {
        return `<svg width="100%" height="28" viewBox="0 0 100 28" preserveAspectRatio="none" style="display:block; background:rgba(0,0,0,0.15); border-radius:4px; margin-top:4px;">
            <line x1="0" y1="24" x2="100" y2="24" stroke="rgba(255,255,255,0.08)" stroke-width="1" stroke-dasharray="2,2"/>
        </svg>`;
    }
    let max = Math.max(...data);
    let min = Math.min(...data);
    let range = Math.max(5, max - min);
    let step = 100 / Math.max(1, data.length - 1);
    let points = data.map((val, i) => {
        let x = (i * step).toFixed(1);
        let y = (26 - ((val - min) / range * 22)).toFixed(1);
        return `${x},${y}`;
    }).join(' ');
    let fillPoints = `0,28 ${points} 100,28`;

    return `<svg width="100%" height="28" viewBox="0 0 100 28" preserveAspectRatio="none" style="display:block; background:rgba(249,115,22,0.08); border:1px solid rgba(249,115,22,0.2); border-radius:4px; margin-top:4px; overflow:hidden;">
        <polygon fill="rgba(249, 115, 22, 0.2)" points="${fillPoints}"/>
        <polyline fill="none" stroke="#f97316" stroke-width="1.5" points="${points}"/>
    </svg>`;
}

function renderCPU(cpu) {
    const el = document.getElementById('content-cpu');
    let html = kvPair(T('Модель'), cpu.model);
    html += kvPair(T('Архитектура'), cpu.architecture || cpu.vendor);
    if (cpu.current_speed_ghz > 0) html += kvPair(T('Частота'), cpu.current_speed_ghz.toFixed(2) + ' GHz');
    if (cpu.package_temp > 0) {
        const color = cpu.package_temp > 80 ? 'color-danger' : (cpu.package_temp > 65 ? 'color-warn' : 'color-ok');
        let tempSpark = makeTempSparklineSVG(window.tempHistoryMap ? window.tempHistoryMap["cpu"] : null);
        html += `<div style="cursor:pointer; margin-top:4px; padding:4px; background:rgba(255,255,255,0.02); border-radius:6px; border:1px solid rgba(249,115,22,0.15);" onclick="openTempGraph('cpu', '${T('Процессор (CPU)')}')">`;
        html += kvPair(T('Температура'), `🔥 ${cpu.package_temp.toFixed(1)} °C <span style="font-size:0.75rem; color:var(--text-secondary);">(График)</span>`, color);
        html += tempSpark;
        html += `</div>`;
    } else {
        html += kvPair(T('Температура'), 'N/A <span style="font-size:0.75rem; color:var(--text-secondary);">(нет доступа к датчику)</span>');
    }
    html += progressBar(cpu.usage_percent, T('Загрузка CPU'), '');
    el.innerHTML = html;
}

function renderMemory(mem) {
    const el = document.getElementById('content-memory');
    let html = '';
    if (mem.spec) {
        html += kvPair(T('Спецификация ОЗУ'), mem.spec, 'color-accent');
    } else {
        if (mem.form_factor || mem.type) {
            let spec = `${mem.form_factor || ''} ${mem.type || ''}`.trim();
            if (mem.speed_mts > 0) spec += `-${mem.speed_mts}`;
            html += kvPair(T('Тип ОЗУ'), spec, 'color-accent');
        }
    }
    html += progressBar(mem.usage_percent, T('Загрузка ОЗУ'), `${formatGB(mem.used_gb)} / ${formatGB(mem.total_gb)}`);
    if (mem.swap_total_gb > 0) {
        const swapPct = (mem.swap_used_gb / mem.swap_total_gb) * 100;
        html += progressBar(swapPct, 'Swap', `${formatGB(mem.swap_used_gb)} / ${formatGB(mem.swap_total_gb)}`);
    }
    if (mem.slots && mem.slots.length > 0) {
        html += `<div style="margin-top: 0.5rem; font-size: 0.8rem;"><strong>${T('Слоты')} (${mem.used_slots || mem.slots.length}/${mem.total_slots || mem.slots.length}):</strong><ul style="padding-left: 1.2rem; margin-top: 0.25rem;">`;
        mem.slots.forEach(s => {
            let details = [s.form_factor, s.type, s.speed_mts ? s.speed_mts + ' MT/s' : '', s.model ? `(${s.model})` : ''].filter(Boolean).join(' ');
            html += `<li>${s.locator || 'Slot'}: ${s.manufacturer || ''} ${s.size_gb} GB ${details}</li>`;
        });
        html += `</ul></div>`;
    }
    el.innerHTML = html;
}

function renderGPU(gpu) {
    if (!gpu || !gpu.gpus || gpu.gpus.length === 0) return;
    document.getElementById('card-gpu').style.display = 'flex';
    const el = document.getElementById('content-gpu');
    let html = '';
    
    const physicalGPUs = gpu.gpus.filter(g => !g.is_virtual);
    const targetGPUs = physicalGPUs.length > 0 ? physicalGPUs : gpu.gpus;

    targetGPUs.forEach(g => {
        let key = "gpu_" + g.name;
        html += `<div class="sub-card">`;
        html += `<div class="sub-card-title">${g.name}</div>`;
        if (g.vram_mb > 0) html += kvPair('VRAM', (g.vram_mb / 1024).toFixed(2) + ' GB');
        if (g.temp_c > 0) {
            const color = g.temp_c > 80 ? 'color-danger' : (g.temp_c > 65 ? 'color-warn' : 'color-ok');
            let tempSpark = makeTempSparklineSVG(window.tempHistoryMap ? window.tempHistoryMap[key] : null);
            html += `<div style="cursor:pointer; margin-top:4px; padding:4px; background:rgba(255,255,255,0.02); border-radius:6px; border:1px solid rgba(249,115,22,0.15);" onclick="openTempGraph('${key}', '${g.name}')">`;
            html += kvPair(T('Температура'), `🔥 ${g.temp_c.toFixed(1)} °C <span style="font-size:0.75rem; color:var(--text-secondary);">(График)</span>`, color);
            html += tempSpark;
            html += `</div>`;
        }
        if (g.gpu_load_pct > 0) html += progressBar(g.gpu_load_pct, T('Загрузка GPU'), '');
        html += `</div>`;
    });

    if (gpu.displays && gpu.displays.length > 0) {
        html += `<div style="margin-top: 0.5rem;">`;
        html += `<button class="btn" style="width: 100%; text-align: center; background: rgba(59, 130, 246, 0.15); border-color: var(--accent-blue);" onclick="openDisplaysModal()">${T('Мониторы и Дисплеи')} (${gpu.displays.length})</button>`;
        html += `</div>`;
    }

    el.innerHTML = html;
}

window.openDisplaysModal = function() {
    if (!currentData || !currentData.gpu || !currentData.gpu.displays) return;
    const modalTitle = document.getElementById('modal-title');
    const modalBody = document.getElementById('modal-body');
    modalTitle.textContent = T('Подключенные Дисплеи и Мониторы');
    
    let html = '<div style="display: flex; flex-direction: column; gap: 0.75rem;">';
    currentData.gpu.displays.forEach(d => {
        let tag = d.is_virtual ? '<span class="color-warn">[Виртуальный/Софтверный]</span>' : '<span class="color-ok">[Физический]</span>';
        html += `<div class="sub-card">`;
        html += `<div class="sub-card-title">${d.name}</div>`;
        if (d.resolution) {
            let res = d.resolution;
            if (d.refresh_rate > 0) res += ` @ ${d.refresh_rate}Hz`;
            html += kvPair(T('Разрешение'), res);
        }
        html += kvPair(T('Тип'), tag);
        if (d.gpu_name) html += kvPair('GPU', d.gpu_name);
        html += `</div>`;
    });
    html += '</div>';
    modalBody.innerHTML = html;

    document.getElementById('modal-overlay').style.display = 'block';
    document.getElementById('modal').style.display = 'flex';
};

function formatBytes(bytes) {
    if (!bytes || bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
}

function makeSparklineSVG(data) {
    if (!data || data.length < 2) {
        return `<svg width="100%" height="32" viewBox="0 0 100 32" preserveAspectRatio="none" style="display:block; background:rgba(0,0,0,0.15); border-radius:4px; margin-top:6px;">
            <line x1="0" y1="28" x2="100" y2="28" stroke="rgba(255,255,255,0.08)" stroke-width="1" stroke-dasharray="2,2"/>
        </svg>`;
    }
    let max = Math.max(...data);
    if (max === 0) max = 1;
    let step = 100 / Math.max(1, data.length - 1);
    let points = data.map((val, i) => {
        let x = (i * step).toFixed(1);
        let y = (30 - (val / max * 26)).toFixed(1);
        return `${x},${y}`;
    }).join(' ');
    let fillPoints = `0,32 ${points} 100,32`;

    return `<svg width="100%" height="32" viewBox="0 0 100 32" preserveAspectRatio="none" style="display:block; background:rgba(0,0,0,0.2); border-radius:4px; margin-top:6px; overflow:hidden;">
        <polygon fill="rgba(59, 130, 246, 0.2)" points="${fillPoints}"/>
        <polyline fill="none" stroke="var(--accent-blue)" stroke-width="1.5" points="${points}"/>
    </svg>`;
}

function renderNetwork(net) {
    if (!net || !net.interfaces) return;
    const el = document.getElementById('content-network');
    document.getElementById('card-network').style.display = 'flex';
    let html = '';
    if (net.dns_servers && net.dns_servers.length > 0) {
        html += `<div style="width: 100%; margin-bottom: 0.5rem;">${kvPair('DNS', net.dns_servers.join(', '))}</div>`;
    }
    html += '<div class="network-grid">';
    net.interfaces.forEach(iface => {
        let hist = window.netHistoryMap ? window.netHistoryMap[iface.name] : null;
        let sparkline = makeSparklineSVG(hist);

        html += `<div class="sub-card" style="cursor:pointer;" onclick="openNetGraph('${iface.name}')">`;
        html += `<div class="sub-card-title">${iface.name} <span class="color-ok">●</span></div>`;
        if (iface.ipv4 && iface.ipv4.length > 0) html += kvPair('IPv4', iface.ipv4[0]);
        if (iface.speed_recv_mbps > 0 || iface.speed_sent_mbps > 0) {
            html += kvPair(T('Скорость'), `<span class="color-accent">↓ ${(iface.speed_recv_mbps || 0).toFixed(2)} Mbps</span> | <span style="color:var(--accent-purple)">↑ ${(iface.speed_sent_mbps || 0).toFixed(2)} Mbps</span>`);
        } else {
            html += kvPair(T('Трафик'), `RX: ${formatBytes(iface.bytes_recv)} | TX: ${formatBytes(iface.bytes_sent)}`);
        }
        html += sparkline;
        html += `</div>`;
    });
    html += '</div>';
    el.innerHTML = html;
}

function renderBattery(bat) {
    if (!bat || !bat.present) return;
    document.getElementById('card-battery').style.display = 'flex';
    const el = document.getElementById('content-battery');
    
    let html = kvPair(T('Статус'), T(bat.status));
    
    let color = bat.charge_pct < 20 ? 'color-danger' : (bat.charge_pct < 50 ? 'color-warn' : 'color-ok');
    let barColor = bat.charge_pct < 20 ? 'bg-danger' : (bat.charge_pct < 50 ? 'bg-warn' : 'bg-success');
    
    html += `
        <div>
            <div class="kv-pair" style="font-size: 0.8rem;">
                <span>${T('Заряд')}</span>
                <span class="${color}">${bat.charge_pct.toFixed(0)}%</span>
            </div>
            <div class="progress-container">
                <div class="progress-bar ${barColor}" style="width: ${bat.charge_pct}%"></div>
            </div>
        </div>
    `;

    if (bat.time_remain) html += kvPair(T('Осталось'), bat.time_remain);
    if (bat.health_pct > 0) html += kvPair(T('Состояние (Здоровье)'), bat.health_pct.toFixed(0) + '%');
    if (bat.cycle_count > 0) html += kvPair(T('Циклов'), bat.cycle_count);
    
    el.innerHTML = html;
}

function renderStorage(storage) {
    if (!storage || !storage.disks) return;
    const el = document.getElementById('content-storage');
    document.getElementById('card-storage').style.display = 'flex';
    let html = '<div class="storage-grid">';
    storage.disks.forEach(d => {
        html += `<div class="sub-card" style="cursor:pointer;" onclick="openSmart('${d.device_name}')">`;
        let tag = d.is_ramdisk ? 'RAM Disk' : (d.media_type || 'Disk');
        html += `<div class="sub-card-title">${d.model} <span style="font-size: 0.75rem; color: var(--text-secondary); float: right;">${tag}</span></div>`;
        
        html += kvPair(T('Объём'), d.size_gb.toFixed(1) + ' GB');
        
        if (d.read_mbps > 0 || d.write_mbps > 0) {
            html += kvPair(T('I/O Скорость'), `<span class="color-accent">R: ${(d.read_mbps || 0).toFixed(1)} MB/s</span> | <span style="color:var(--accent-purple)">W: ${(d.write_mbps || 0).toFixed(1)} MB/s</span>`);
        }

        if (d.health) {
            let hColor = d.health === 'OK' ? 'color-ok' : (d.health === 'Unknown' ? '' : 'color-danger');
            html += kvPair('SMART Health', d.health, hColor);
        }
        if (d.temp_c > 0) {
            let key = "disk_" + d.model;
            let tColor = d.temp_c > 55 ? 'color-danger' : (d.temp_c > 45 ? 'color-warn' : 'color-ok');
            let tempSpark = makeTempSparklineSVG(window.tempHistoryMap ? window.tempHistoryMap[key] : null);
            html += `<div style="cursor:pointer; margin-top:4px; padding:4px; background:rgba(255,255,255,0.02); border-radius:6px; border:1px solid rgba(249,115,22,0.15);" onclick="event.stopPropagation(); openTempGraph('${key}', '${d.model}')">`;
            html += kvPair(T('Температура'), `🔥 ${d.temp_c.toFixed(1)} °C <span style="font-size:0.75rem; color:var(--text-secondary);">(График)</span>`, tColor);
            html += tempSpark;
            html += `</div>`;
        }

        if (d.partitions && d.partitions.length > 0) {
            html += `<div style="margin-top: 0.5rem;">`;
            d.partitions.forEach(p => {
                html += progressBar(p.used_pct, p.mount_point, `${p.free_gb.toFixed(1)} GB free`);
            });
            html += `</div>`;
        }

        if (d.smart && d.smart.length > 0) {
            let warnings = d.smart.filter(ind => ind.is_warning);
            if (warnings.length > 0) {
                html += `<div class="alert-box"><strong>Внимание!</strong> ${warnings.length} показателей вышли за пределы:<ul>`;
                warnings.forEach(w => { html += `<li>${w.name}: ${w.raw_value}</li>`; });
                html += `</ul></div>`;
            }
        }

        html += `</div>`;
    });
    html += '</div>';
    el.innerHTML = html;
}

function renderProcesses(proc) {
    if (!proc || !proc.processes) return;
    const el = document.getElementById('content-processes');
    document.getElementById('card-processes').style.display = 'flex';
    let html = `<div class="table-container" style="max-height: 400px; overflow-y: auto;"><table>`;
    html += `<thead><tr><th>PID</th><th>${T('Имя')}</th><th class="text-right">CPU %</th><th class="text-right">RAM %</th></tr></thead><tbody>`;
    
    let procs = proc.processes;
    if (!showAllProcesses && procs.length > 10) {
        procs = procs.slice(0, 10);
    }

    procs.forEach(p => {
        html += `<tr class="hover-row" style="cursor:pointer;" onclick="confirmKill(${p.pid})">
            <td>${p.pid}</td>
            <td>${p.name.length > 20 ? p.name.substring(0, 20) + '...' : p.name}</td>
            <td class="text-right">${p.cpu_pct.toFixed(1)}</td>
            <td class="text-right">${p.mem_pct.toFixed(1)}</td>
        </tr>`;
    });
    
    html += `</tbody></table></div>`;
    let btnText = showAllProcesses ? T('Скрыть') : T('Показать все');
    html += `<div style="display: flex; justify-content: space-between; align-items: center; margin-top: 0.5rem; font-size: 0.8rem;">
        <button class="btn" style="cursor: pointer; padding: 0.2rem 0.5rem; border-color: transparent; background: rgba(255,255,255,0.1);" onclick="toggleProcesses()">${btnText}</button>
        <div>${T('Всего процессов')}: ${proc.total_count}</div>
    </div>`;
    el.innerHTML = html;
}

function openModal(title, contentHTML) {
    document.getElementById('modal-title').textContent = title;
    document.getElementById('modal-body').innerHTML = contentHTML;
    document.getElementById('modal-overlay').style.display = 'block';
    document.getElementById('modal').style.display = 'flex';
}

window.closeModal = function() {
    document.getElementById('modal-overlay').style.display = 'none';
    document.getElementById('modal').style.display = 'none';
    window.activeNetGraph = null;
    if (window.netChartInstance) {
        window.netChartInstance.destroy();
        window.netChartInstance = null;
    }
};

window.confirmKill = async function(pid) {
    if(!confirm(T("Завершить процесс") + " " + pid + "?")) return;
    try {
        const res = await fetch('/api/process/terminate?pid=' + pid, {method: 'POST'});
        if(res.ok) {
            closeModal();
            fetchData();
        } else {
            alert(await res.text());
        }
    } catch(e) {
        alert(e);
    }
};

window.openSmart = async function(device) {
    openModal("SMART: " + device, '<div style="text-align:center;">' + T("Загрузка...") + '</div>');
    try {
        const res = await fetch('/api/smart?device=' + encodeURIComponent(device));
        const text = await res.text();
        document.getElementById('modal-body').innerHTML = '<div class="pre-box">' + text + '</div>';
    } catch(e) {
        document.getElementById('modal-body').innerHTML = '<div class="color-danger">' + e + '</div>';
    }
};

window.openNetGraph = async function(iface) {
    if (window.netChartInstance) {
        window.netChartInstance.destroy();
        window.netChartInstance = null;
    }
    window.activeNetGraph = iface;
    openModal(T("Активность сети") + ": " + iface, '<div style="padding:20px; text-align:center; color:var(--text-secondary);">' + T("Загрузка...") + '</div>');
    
    window.netHistoryMap = window.netHistoryMap || {};
    if (!window.netHistoryMap[iface]) {
        window.netHistoryMap[iface] = Array(30).fill(0);
    }

    try {
        const res = await fetch('/api/network/history');
        if (res.ok) {
            const hist = await res.json();
            if (hist && hist[iface]) {
                window.netHistoryMap[iface] = hist[iface];
            }
        }
    } catch(e) {
        console.warn("API /api/network/history unavailable, using real-time polling data:", e);
    }
    
    window.renderNetGraphModal();
};

window.renderNetGraphModal = function() {
    if (!window.activeNetGraph) return;
    const iface = window.activeNetGraph;
    window.netHistoryMap = window.netHistoryMap || {};
    const data = window.netHistoryMap[iface] || [0];
    const latest = data[data.length - 1] || 0;
    let max = Math.max(...data);
    if (max === 0) max = 1;

    const modalBody = document.getElementById('modal-body');
    let canvas = document.getElementById('net-chart-canvas');

    if (!canvas) {
        let html = `<div style="display:flex; justify-content:space-between; align-items:center; font-size:0.9rem; margin-bottom:12px;">
            <div><b>${T("Текущая скорость")}:</b> <span id="net-curr-speed" class="color-accent" style="font-weight:600;">${latest.toFixed(2)} Mbps</span></div>
            <div><b>${T("Пиковая скорость")}:</b> <span id="net-peak-speed" style="font-weight:600;">${max.toFixed(2)} Mbps</span></div>
        </div>
        <div style="position:relative; width:100%; height:250px; background:rgba(0,0,0,0.2); border:1px solid var(--card-border); border-radius:8px; padding:8px; box-sizing:border-box;">
            <canvas id="net-chart-canvas"></canvas>
        </div>`;
        modalBody.innerHTML = html;
        canvas = document.getElementById('net-chart-canvas');
    } else {
        const currEl = document.getElementById('net-curr-speed');
        const peakEl = document.getElementById('net-peak-speed');
        if (currEl) currEl.innerText = latest.toFixed(2) + ' Mbps';
        if (peakEl) peakEl.innerText = max.toFixed(2) + ' Mbps';
    }

    const labels = data.map((_, i) => {
        let secs = (data.length - 1 - i) * 2;
        return secs === 0 ? T("Сейчас") : `-${secs}s`;
    });

    if (typeof Chart !== 'undefined' && canvas) {
        if (window.netChartInstance) {
            window.netChartInstance.data.labels = labels;
            window.netChartInstance.data.datasets[0].data = data;
            window.netChartInstance.update('none');
        } else {
            const ctx = canvas.getContext('2d');
            const gradient = ctx.createLinearGradient(0, 0, 0, 200);
            gradient.addColorStop(0, 'rgba(59, 130, 246, 0.4)');
            gradient.addColorStop(1, 'rgba(59, 130, 246, 0.0)');

            window.netChartInstance = new Chart(ctx, {
                type: 'line',
                data: {
                    labels: labels,
                    datasets: [{
                        label: 'Скорость (Mbps)',
                        data: data,
                        borderColor: '#3b82f6',
                        borderWidth: 2,
                        backgroundColor: gradient,
                        fill: true,
                        tension: 0.4,
                        pointRadius: 2,
                        pointHoverRadius: 6,
                        pointBackgroundColor: '#60a5fa'
                    }]
                },
                options: {
                    responsive: true,
                    maintainAspectRatio: false,
                    animation: false,
                    plugins: {
                        legend: { display: false },
                        tooltip: {
                            callbacks: {
                                label: function(context) {
                                    return context.parsed.y.toFixed(2) + ' Mbps';
                                }
                            }
                        }
                    },
                    scales: {
                        x: {
                            grid: { color: 'rgba(255, 255, 255, 0.05)' },
                            ticks: { color: '#94a3b8', font: { size: 10 } }
                        },
                        y: {
                            beginAtZero: true,
                            grid: { color: 'rgba(255, 255, 255, 0.05)' },
                            ticks: { 
                                color: '#94a3b8', 
                                font: { size: 10 },
                                callback: function(value) { return value.toFixed(1) + ' Mbps'; }
                            }
                        }
                    }
                }
            });
        }
    }
};

window.openDiagnosticModal = function() {
    document.getElementById('modal-title').innerText = T("Диагностика системы и сенсоров");
    document.getElementById('modal-body').innerHTML = `<div style="padding:30px; text-align:center; color:var(--warn); font-size:1.05rem;">⌛ ${T("Идёт глубокое сканирование сенсоров, прав и подсистем...")}</div>`;
    document.getElementById('modal-overlay').style.display = 'block';
    document.getElementById('modal').style.display = 'flex';

    fetch('/api/diagnostic')
        .then(r => r.json())
        .then(data => {
            let html = `<div style="display:flex; flex-direction:column; gap:12px;">`;
            html += `<div style="padding:10px 14px; background:rgba(255,255,255,0.03); border:1px solid var(--card-border); border-radius:8px; font-size:0.9rem; display:flex; gap:15px; flex-wrap:wrap; align-items:center;">
                <div><b>ОС:</b> ${data.os} (${data.kernel})</div>
                <div><b>Хост:</b> ${data.hostname}</div>
                <div><b>Права:</b> ${data.is_admin ? '<span class="color-ok" style="font-weight:600;">Администратор / Root (Полный доступ)</span>' : '<span class="color-warn" style="font-weight:600;">Обычный пользователь (Ограниченный доступ)</span>'}</div>
            </div>`;

            data.reports.forEach(rep => {
                html += `<div class="diag-card">
                    <div class="diag-title">${rep.component_name}</div>`;
                rep.checks.forEach(ch => {
                    let statusClass = '';
                    let badge = '<span class="badge" style="background:var(--ok); color:#000; font-weight:700;">OK</span>';
                    if (ch.status === 'WARN') {
                        badge = '<span class="badge" style="background:var(--warn); color:#000; font-weight:700;">ВНИМАНИЕ</span>';
                        statusClass = 'warn';
                    }
                    if (ch.status === 'FAIL') {
                        badge = '<span class="badge" style="background:var(--danger); color:#fff; font-weight:700;">ОШИБКА</span>';
                        statusClass = 'fail';
                    }

                    html += `<div class="diag-item ${statusClass}">
                        <div class="diag-item-header">
                            <span>${ch.name}</span>
                            ${badge}
                        </div>`;
                    if (ch.value) html += `<div style="font-size:0.85rem; color:var(--text-secondary); margin-top:3px;">${ch.value}</div>`;
                    if (ch.error_message) html += `<div class="diag-symptom"><b>Симптом:</b> ${ch.error_message}</div>`;
                    if (ch.root_cause) html += `<div class="diag-cause"><b>Причина:</b> ${ch.root_cause}</div>`;
                    if (ch.recommendation) html += `<div class="diag-solution"><b>Решение:</b> ${ch.recommendation.replace(/\n/g, '<br>')}</div>`;
                    html += `</div>`;
                });
                html += `</div>`;
            });
            html += `</div>`;
            document.getElementById('modal-body').innerHTML = html;
        })
        .catch(err => {
            document.getElementById('modal-body').innerHTML = `<div style="color:var(--danger); padding:20px;">Не удалось загрузить отчет диагностики: ${err}</div>`;
        });
};

window.openTempGraph = async function(key, title) {
    if (window.tempChartInstance) {
        window.tempChartInstance.destroy();
        window.tempChartInstance = null;
    }
    window.activeTempGraph = key;
    window.activeTempTitle = title;
    openModal(T("График температуры") + ": " + title, '<div style="padding:20px; text-align:center; color:var(--text-secondary);">' + T("Загрузка...") + '</div>');
    
    window.tempHistoryMap = window.tempHistoryMap || {};
    if (!window.tempHistoryMap[key]) {
        window.tempHistoryMap[key] = [0];
    }

    try {
        const res = await fetch('/api/temp/history');
        if (res.ok) {
            const hist = await res.json();
            if (hist && hist[key]) {
                window.tempHistoryMap[key] = hist[key];
            }
        }
    } catch(e) {
        console.warn("API /api/temp/history unavailable, using real-time polling data:", e);
    }
    
    window.renderTempGraphModal();
};

window.renderTempGraphModal = function() {
    if (!window.activeTempGraph) return;
    const key = window.activeTempGraph;
    window.tempHistoryMap = window.tempHistoryMap || {};
    const data = window.tempHistoryMap[key] || [0];
    const latest = data[data.length - 1] || 0;
    let max = Math.max(...data);
    let min = Math.min(...data);

    const modalBody = document.getElementById('modal-body');
    let canvas = document.getElementById('temp-chart-canvas');

    if (!canvas) {
        let html = `<div style="display:flex; justify-content:space-between; align-items:center; font-size:0.9rem; margin-bottom:12px;">
            <div><b>${T("Текущая темп.")}:</b> <span id="temp-curr-val" style="color:#f97316; font-weight:600;">🔥 ${latest.toFixed(1)} °C</span></div>
            <div><b>${T("Макс")}:</b> <span id="temp-peak-val" style="font-weight:600; color:var(--danger);">${max.toFixed(1)} °C</span></div>
            <div><b>${T("Мин")}:</b> <span id="temp-min-val" style="font-weight:600; color:var(--ok);">${min.toFixed(1)} °C</span></div>
        </div>
        <div style="position:relative; width:100%; height:250px; background:rgba(0,0,0,0.2); border:1px solid var(--card-border); border-radius:8px; padding:8px; box-sizing:border-box;">
            <canvas id="temp-chart-canvas"></canvas>
        </div>`;
        modalBody.innerHTML = html;
        canvas = document.getElementById('temp-chart-canvas');
    } else {
        const currEl = document.getElementById('temp-curr-val');
        const peakEl = document.getElementById('temp-peak-val');
        const minEl = document.getElementById('temp-min-val');
        if (currEl) currEl.innerText = '🔥 ' + latest.toFixed(1) + ' °C';
        if (peakEl) peakEl.innerText = max.toFixed(1) + ' °C';
        if (minEl) minEl.innerText = min.toFixed(1) + ' °C';
    }

    const labels = data.map((_, i) => {
        let secs = (data.length - 1 - i) * 2;
        return secs === 0 ? T("Сейчас") : `-${secs}s`;
    });

    if (typeof Chart !== 'undefined' && canvas) {
        if (window.tempChartInstance) {
            window.tempChartInstance.data.labels = labels;
            window.tempChartInstance.data.datasets[0].data = data;
            window.tempChartInstance.update('none');
        } else {
            const ctx = canvas.getContext('2d');
            const gradient = ctx.createLinearGradient(0, 0, 0, 200);
            gradient.addColorStop(0, 'rgba(249, 115, 22, 0.4)');
            gradient.addColorStop(1, 'rgba(249, 115, 22, 0.0)');

            window.tempChartInstance = new Chart(ctx, {
                type: 'line',
                data: {
                    labels: labels,
                    datasets: [{
                        label: 'Температура (°C)',
                        data: data,
                        borderColor: '#f97316',
                        borderWidth: 2,
                        backgroundColor: gradient,
                        fill: true,
                        tension: 0.4,
                        pointRadius: 2,
                        pointHoverRadius: 6,
                        pointBackgroundColor: '#ef4444'
                    }]
                },
                options: {
                    responsive: true,
                    maintainAspectRatio: false,
                    animation: false,
                    plugins: {
                        legend: { display: false },
                        tooltip: {
                            callbacks: {
                                label: function(context) {
                                    return context.parsed.y.toFixed(1) + ' °C';
                                }
                            }
                        }
                    },
                    scales: {
                        x: {
                            grid: { color: 'rgba(255, 255, 255, 0.05)' },
                            ticks: { color: '#94a3b8', font: { size: 10 } }
                        },
                        y: {
                            grid: { color: 'rgba(255, 255, 255, 0.05)' },
                            ticks: { 
                                color: '#94a3b8', 
                                font: { size: 10 },
                                callback: function(value) { return value.toFixed(0) + ' °C'; }
                            }
                        }
                    }
                }
            });
        }
    }
};
