import { useEffect, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useAuthStore } from '../../stores/auth';
import { Terminal } from '@xterm/xterm';
import { FitAddon } from '@xterm/addon-fit';
import { WebLinksAddon } from '@xterm/addon-web-links';
import '@xterm/xterm/css/xterm.css';
import { Plus, X } from 'lucide-react';

interface TerminalTab {
  id: string;
  name: string;
  terminal: Terminal;
  fitAddon: FitAddon;
  ws: WebSocket | null;
}

export default function TerminalPage() {
  useTranslation();
  const { accessToken } = useAuthStore();
  const [tabs, setTabs] = useState<TerminalTab[]>([]);
  const [activeTab, setActiveTab] = useState<string>('');
  const termContainerRef = useRef<HTMLDivElement>(null);

  const createTab = (name?: string) => {
    const id = `term-${Date.now()}`;
    const tabName = name || `Terminal ${tabs.length + 1}`;

    const terminal = new Terminal({
      cursorBlink: true,
      fontSize: 14,
      fontFamily: 'Menlo, Monaco, "Courier New", monospace',
      allowProposedApi: true,
      theme: {
        background: '#1a1b26',
        foreground: '#c0caf5',
        cursor: '#c0caf5',
        selectionBackground: '#33467c',
        black: '#15161e',
        red: '#f7768e',
        green: '#9ece6a',
        yellow: '#e0af68',
        blue: '#7aa2f7',
        magenta: '#bb9af7',
        cyan: '#7dcfff',
        white: '#a9b1d6',
      },
      scrollback: 5000,
    });

    const fitAddon = new FitAddon();
    terminal.loadAddon(fitAddon);
    terminal.loadAddon(new WebLinksAddon());

    // Connect WebSocket
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const host = window.location.host;
    const url = `${protocol}//${host}/api/v1/terminal/ws?token=${accessToken}`;
    const ws = new WebSocket(url);

    ws.binaryType = 'arraybuffer';

    ws.onopen = () => {
      terminal.writeln('\x1b[32mConnected.\x1b[0m');
      // Send initial resize
      const dims = fitAddon.proposeDimensions();
      if (dims) {
        ws.send(JSON.stringify({ type: 'resize', cols: dims.cols, rows: dims.rows }));
      }
    };

    ws.onmessage = (event) => {
      if (event.data instanceof ArrayBuffer) {
        terminal.write(new Uint8Array(event.data));
      } else {
        terminal.write(event.data);
      }
    };

    ws.onclose = () => {
      terminal.writeln('\r\n\x1b[31mDisconnected.\x1b[0m');
    };

    // Copy on select
    terminal.onSelectionChange(() => {
      const sel = terminal.getSelection();
      if (sel) {
        navigator.clipboard.writeText(sel).catch(() => {});
      }
    });

    // Paste with Ctrl+V
    terminal.attachCustomKeyEventHandler((e) => {
      if (e.ctrlKey && e.key === 'v' && e.type === 'keydown') {
        navigator.clipboard.readText().then(text => {
          if (ws.readyState === WebSocket.OPEN) ws.send(text);
        }).catch(() => {});
        return false;
      }
      return true;
    });

    // Send input to WebSocket
    terminal.onData((data) => {
      if (ws.readyState === WebSocket.OPEN) {
        ws.send(data);
      }
    });

    const newTab: TerminalTab = { id, name: tabName, terminal, fitAddon, ws };

    setTabs(prev => [...prev, newTab]);
    setActiveTab(id);

    // Mount terminal after state update
    setTimeout(() => {
      if (termContainerRef.current) {
        // Clear container for new active terminal
        const container = document.getElementById(`term-container-${id}`);
        if (container) {
          terminal.open(container);
          fitAddon.fit();
        }
      }
    }, 50);
  };

  const closeTab = (id: string) => {
    const tab = tabs.find(t => t.id === id);
    if (tab) {
      tab.ws?.close();
      tab.terminal.dispose();
    }
    setTabs(prev => {
      const next = prev.filter(t => t.id !== id);
      if (activeTab === id && next.length > 0) {
        setActiveTab(next[next.length - 1].id);
      }
      return next;
    });
  };

  // Create first tab on mount
  useEffect(() => {
    if (tabs.length === 0) {
      createTab();
    }
    return () => {
      tabs.forEach(t => {
        t.ws?.close();
        t.terminal.dispose();
      });
    };
  }, []);

  // Refit on active tab change
  useEffect(() => {
    const tab = tabs.find(t => t.id === activeTab);
    if (tab) {
      setTimeout(() => {
        const container = document.getElementById(`term-container-${tab.id}`);
        if (container && !container.querySelector('.xterm')) {
          tab.terminal.open(container);
        }
        tab.fitAddon.fit();
        tab.terminal.focus();
      }, 50);
    }
  }, [activeTab]);

  // Handle window resize
  useEffect(() => {
    const handleResize = () => {
      const tab = tabs.find(t => t.id === activeTab);
      if (tab) tab.fitAddon.fit();
    };
    window.addEventListener('resize', handleResize);
    return () => window.removeEventListener('resize', handleResize);
  }, [activeTab, tabs]);

  return (
    <div className="h-full flex flex-col">
      {/* Tab bar */}
      <div className="flex items-center bg-gray-800 rounded-t-xl overflow-x-auto">
        {tabs.map(tab => (
          <div
            key={tab.id}
            onClick={() => setActiveTab(tab.id)}
            className={`flex items-center gap-2 px-4 py-2 cursor-pointer text-sm border-r border-gray-700 ${
              activeTab === tab.id ? 'bg-gray-900 text-white' : 'bg-gray-800 text-gray-400 hover:text-white'
            }`}
          >
            <span>{tab.name}</span>
            <button
              onClick={(e) => { e.stopPropagation(); closeTab(tab.id); }}
              className="hover:text-red-400"
            >
              <X size={12} />
            </button>
          </div>
        ))}
        <button onClick={() => createTab()} className="px-3 py-2 text-gray-400 hover:text-white">
          <Plus size={16} />
        </button>
      </div>

      {/* Terminal containers */}
      <div ref={termContainerRef} className="flex-1 bg-[#1a1b26] rounded-b-xl overflow-hidden relative">
        {tabs.map(tab => (
          <div
            key={tab.id}
            id={`term-container-${tab.id}`}
            className={`absolute inset-0 ${activeTab === tab.id ? '' : 'hidden'}`}
          />
        ))}
        {tabs.length === 0 && (
          <div className="flex items-center justify-center h-full text-gray-500">
            Click + to open a terminal
          </div>
        )}
      </div>
    </div>
  );
}
