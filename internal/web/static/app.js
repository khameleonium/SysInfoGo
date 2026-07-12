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
    if (data.cpu) renderCPU(data.cpu);
    if (data.memory) renderMemory(data.memory);
    if (data.gpu) renderGPU(data.gpu);
    if (data.network) renderNetwork(data.network);
    if (data.battery) renderBattery(data.battery);
    if (data.storage) renderStorage(data.storage);
    if (data.processes) renderProcesses(data.processes);
    
    if (data.network && window.activeNetGraph) {
        let iface = data.network.interfaces.find(i => i.name === window.activeNetGraph);
        if (iface) {
            let val = (iface.speed_recv_mbps || 0) + (iface.speed_sent_mbps || 0);
            if (!window.activeNetGraphData) window.activeNetGraphData = Array(50).fill(0);
            window.activeNetGraphData.push(val);
            if (window.activeNetGraphData.length > 50) window.activeNetGraphData.shift();
            if (window.renderNetGraphModal) window.renderNetGraphModal();
        }
    }
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
    let html = kvPair(T('Хост'), summary.hostname);
    if (summary.virtualization) {
        html += kvPair(T('Виртуализация'), summary.virtualization, 'color-accent');
    }
    if (summary.motherboard) {
        html += kvPair(T('Материнская плата'), summary.motherboard);
    }
    html += kvPair(T('Ядро'), summary.kernel);
    html += kvPair(T('Процессор'), summary.cpu_model);
    if (summary.boot_time) html += kvPair(T('Запуск'), summary.boot_time);
    html += kvPair('RAM', formatGB(summary.ram_total_gb));
    
    el.innerHTML = html;
}

function renderCPU(cpu) {
    const el = document.getElementById('content-cpu');
    document.getElementById('card-cpu').style.display = 'flex';
    let html = kvPair(T('Модель'), cpu.model);
    html += kvPair(T('Архитектура'), cpu.architecture || cpu.vendor);
    if (cpu.current_speed_ghz > 0) html += kvPair(T('Частота'), cpu.current_speed_ghz.toFixed(2) + ' GHz');
    if (cpu.package_temp > 0) {
        const color = cpu.package_temp > 80 ? 'color-danger' : (cpu.package_temp > 65 ? 'color-warn' : 'color-ok');
        html += kvPair(T('Температура'), cpu.package_temp.toFixed(1) + ' °C', color);
    }
    html += progressBar(cpu.usage_percent, T('Общая загрузка'), '');
    el.innerHTML = html;
}

function renderMemory(mem) {
    const el = document.getElementById('content-memory');
    document.getElementById('card-memory').style.display = 'flex';
    let html = progressBar(mem.usage_percent, T('ОЗУ'), `${formatGB(mem.used_gb)} / ${formatGB(mem.total_gb)}`);
    if (mem.swap_total_gb > 0) {
        const swapPct = (mem.swap_used_gb / mem.swap_total_gb) * 100;
        html += progressBar(swapPct, 'Swap', `${formatGB(mem.swap_used_gb)} / ${formatGB(mem.swap_total_gb)}`);
    }
    el.innerHTML = html;
}

function renderGPU(gpu) {
    if (!gpu || !gpu.gpus || gpu.gpus.length === 0) return;
    document.getElementById('card-gpu').style.display = 'flex';
    const el = document.getElementById('content-gpu');
    let html = '';
    gpu.gpus.forEach(g => {
        html += `<div class="sub-card">`;
        html += `<div class="sub-card-title">${g.name}</div>`;
        if (g.vram_mb > 0) html += kvPair('VRAM', (g.vram_mb / 1024).toFixed(2) + ' GB');
        if (g.temp_c > 0) {
            const color = g.temp_c > 80 ? 'color-danger' : (g.temp_c > 65 ? 'color-warn' : 'color-ok');
            html += kvPair(T('Температура'), g.temp_c.toFixed(1) + ' °C', color);
        }
        if (g.gpu_load_pct > 0) html += progressBar(g.gpu_load_pct, T('Загрузка GPU'), '');
        html += `</div>`;
    });
    el.innerHTML = html;
}

function formatBytes(bytes) {
    if (!bytes || bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
}

function renderNetwork(net) {
    if (!net || !net.interfaces) return;
    const el = document.getElementById('content-network');
    document.getElementById('card-network').style.display = 'flex';
    let html = '';
    if (net.dns_servers && net.dns_servers.length > 0) {
        html += kvPair('DNS', net.dns_servers.join(', '));
    }
    net.interfaces.forEach(iface => {
        html += `<div class="sub-card" style="cursor:pointer;" onclick="openNetGraph('${iface.name}')">`;
        html += `<div class="sub-card-title">${iface.name} <span class="color-ok">●</span></div>`;
        if (iface.ipv4 && iface.ipv4.length > 0) html += kvPair('IPv4', iface.ipv4[0]);
        if (iface.speed_recv_mbps > 0 || iface.speed_sent_mbps > 0) {
            html += kvPair(T('Скорость'), `<span class="color-accent">↓ ${(iface.speed_recv_mbps || 0).toFixed(2)} Mbps</span> | <span style="color:var(--accent-purple)">↑ ${(iface.speed_sent_mbps || 0).toFixed(2)} Mbps</span>`);
        } else {
            html += kvPair(T('Трафик'), `RX: ${formatBytes(iface.bytes_recv)} | TX: ${formatBytes(iface.bytes_sent)}`);
        }
        html += `</div>`;
    });
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
    let html = '';
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
            let tColor = d.temp_c > 55 ? 'color-danger' : (d.temp_c > 45 ? 'color-warn' : 'color-ok');
            html += kvPair(T('Температура'), d.temp_c.toFixed(1) + ' °C', tColor);
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
    window.activeNetGraph = iface;
    openModal(T("Активность сети") + ": " + iface, '<div style="text-align:center;">' + T("Загрузка...") + '</div>');
    try {
        const res = await fetch('/api/network/history');
        const hist = await res.json();
        window.activeNetGraphData = hist[iface] || [0];
        window.renderNetGraphModal();
    } catch(e) {
        document.getElementById('modal-body').innerHTML = '<div class="color-danger">' + e + '</div>';
    }
};

window.renderNetGraphModal = function() {
    if (!window.activeNetGraph) return;
    const data = window.activeNetGraphData;
    if (!data || data.length === 0) {
        document.getElementById('modal-body').innerHTML = '<div>No data</div>';
        return;
    }
    
    let max = Math.max(...data);
    if(max === 0) max = 1;
    
    let svg = '<svg width="100%" height="200" viewBox="0 0 100 200" preserveAspectRatio="none" style="background: rgba(0,0,0,0.2); border-radius:8px; margin-top:1rem;">';
    let step = 100 / Math.max(1, data.length - 1);
    
    let points = data.map((val, i) => {
        let x = i * step;
        let y = 200 - (val / max * 180);
        return x + ',' + y;
    }).join(' ');
    
    svg += '<polyline fill="none" stroke="var(--accent-blue)" stroke-width="2" points="' + points + '"/>';
    svg += '</svg>';
    
    let html = '<div style="font-size:0.8rem; color:var(--text-secondary);">' + T("Максимум") + ': ' + max.toFixed(2) + ' Mbps</div>';
    html += svg;
    document.getElementById('modal-body').innerHTML = html;
};
