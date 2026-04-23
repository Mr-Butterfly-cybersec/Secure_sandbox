document.addEventListener('DOMContentLoaded', () => {
    const runBtn = document.getElementById('run-btn');
    const outputDiv = document.getElementById('output');
    const historyBody = document.getElementById('history-body');

    runBtn.addEventListener('click', async () => {
        const lang = document.getElementById('language').value;
        const code = document.getElementById('code').value;

        if (!code.trim()) return;

        runBtn.disabled = true;
        runBtn.innerText = 'WAIT';
        document.getElementById('res-status').innerText = 'PROCESSING';
        document.getElementById('res-status').className = 'text-[10px] font-bold uppercase tracking-widest text-blue-500/70';

        try {
            const res = await fetch('/execute', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ language: lang, code: code })
            });

            const data = await res.json();
            
            outputDiv.innerText = data.stdout + (data.stderr ? '\nSYSTEM_ERR:\n' + data.stderr : '');
            
            let statusText = 'COMPLETED';
            let statusClass = 'text-green-500/80';
            
            if (data.timed_out) {
                statusText = 'TIMEOUT';
                statusClass = 'text-yellow-500/80';
            } else if (data.terminated) {
                statusText = 'VIOLATION';
                statusClass = 'text-red-500/80';
            }

            document.getElementById('res-status').innerText = statusText;
            document.getElementById('res-status').className = `text-[10px] font-bold uppercase tracking-widest ${statusClass}`;

            fetchLogs();
        } catch (err) {
            outputDiv.innerText = 'CONNECTION_ERR';
        } finally {
            runBtn.disabled = false;
            runBtn.innerText = 'EXECUTE';
        }
    });

    async function fetchLogs() {
        try {
            const res = await fetch('/logs');
            const logs = await res.json();
            
            updateStats(logs);
            updateHistory(logs);
        } catch (err) {
            console.error('LOG_FETCH_ERR', err);
        }
    }

    function updateStats(logs) {
        let total = logs.length;
        let triggers = logs.filter(l => l.status.includes('Terminated') || l.status === 'TimedOut').length;

        document.getElementById('stat-total').innerText = total;
        document.getElementById('stat-traps').innerText = triggers;
    }

    function updateHistory(logs) {
        historyBody.innerHTML = '';
        logs.reverse().slice(0, 50).forEach(log => {
            const tr = document.createElement('tr');
            tr.className = 'border-b border-neutral-900/30 hover:bg-neutral-900/40 transition-all duration-200';
            
            let statusText = log.status.toUpperCase();
            let statusColor = 'text-neutral-600';
            
            if (statusText.includes('TERMINATED')) {
                statusColor = 'text-red-500/60';
                statusText = 'VIOLATION';
            } else if (statusText === 'TIMEDOUT') {
                statusColor = 'text-yellow-500/60';
                statusText = 'TIMEOUT';
            } else if (statusText === 'SUCCESS') {
                statusText = 'COMPLETED';
            }

            const timeStr = new Date(log.timestamp).toLocaleTimeString([], { hour12: false, hour: '2-digit', minute: '2-digit' });

            tr.innerHTML = `
                <td class="py-3 px-3 tabular-nums text-neutral-700 font-mono text-[10px]">${timeStr}</td>
                <td class="py-3 px-3 text-neutral-400 font-medium">${log.language.split('@')[0]}</td>
                <td class="py-3 px-3 ${statusColor} font-bold tracking-tighter text-[9px] uppercase">${statusText}</td>
                <td class="py-3 px-3 text-right text-neutral-700 font-mono text-[10px]">${log.duration.split('.')[0]}ms</td>
            `;
            historyBody.appendChild(tr);
        });
    }

    fetchLogs();
    setInterval(fetchLogs, 5000);
});
