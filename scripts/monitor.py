#!/usr/bin/env python3
"""
miau Dev Monitor - TUI para acompanhar processos e logs
"""

import os
import select
import subprocess
import sys
import termios
import time
import tty
from pathlib import Path

try:
    from rich.console import Console
    from rich.layout import Layout
    from rich.live import Live
    from rich.panel import Panel
    from rich.table import Table
    from rich.text import Text
    from collections import deque
except ImportError:
    print("\n[ERRO] Biblioteca 'rich' não encontrada.")
    print("Instale com: sudo dnf install python3-rich")
    print("         ou: pip install --user rich")
    sys.exit(1)

# Configuração
LOG_FILE = os.environ.get("MIAU_LOG", "/tmp/miau-dev.log")
ERROR_FILE = "/tmp/miau-last-error.txt"
MAX_LOG_LINES = 18
REFRESH_INTERVAL = 2

# PIDs passados via ambiente
WAILS_PID = os.environ.get("WAILS_PID", "")

# Estado global
last_status_message = ""
last_status_time = 0
has_error = False


def get_process_stats(pid: str) -> dict | None:
    """Obtém stats de um processo via ps"""
    if not pid:
        return None
    try:
        result = subprocess.run(
            ["ps", "-p", pid, "-o", "%cpu,%mem,rss", "--no-headers"],
            capture_output=True,
            text=True,
        )
        if result.returncode != 0:
            return None
        parts = result.stdout.strip().split()
        if len(parts) >= 3:
            return {
                "cpu": float(parts[0]),
                "mem": float(parts[1]),
                "rss_mb": int(parts[2]) / 1024,
            }
    except:
        pass
    return None


def find_process_pid(pattern: str) -> str:
    """Encontra PID de um processo pelo pattern"""
    try:
        result = subprocess.run(
            ["pgrep", "-f", pattern],
            capture_output=True,
            text=True,
        )
        if result.returncode == 0:
            pids = result.stdout.strip().split("\n")
            if pids and pids[0]:
                return pids[0]
    except:
        pass
    return ""


def get_system_stats() -> dict:
    """Obtém stats do sistema"""
    try:
        # CPU total
        result = subprocess.run(
            ["ps", "-eo", "%cpu", "--no-headers"], capture_output=True, text=True
        )
        cpu_total = sum(float(x) for x in result.stdout.split() if x)

        # Memória
        result = subprocess.run(["free", "-m"], capture_output=True, text=True)
        for line in result.stdout.split("\n"):
            if line.startswith("Mem:"):
                parts = line.split()
                total = int(parts[1])
                used = int(parts[2])
                return {
                    "cpu": cpu_total,
                    "mem_used": used,
                    "mem_total": total,
                    "mem_pct": (used / total) * 100,
                }
    except:
        pass
    return {"cpu": 0, "mem_used": 0, "mem_total": 0, "mem_pct": 0}


def get_db_stats() -> dict:
    """Obtém stats do banco SQLite"""
    db_path = Path.home() / ".config" / "miau" / "data" / "miau.db"
    try:
        if db_path.exists():
            size_mb = db_path.stat().st_size / (1024 * 1024)
            return {"exists": True, "size_mb": size_mb}
    except:
        pass
    return {"exists": False, "size_mb": 0}


def read_log_tail(path: str, lines: int = MAX_LOG_LINES) -> list[str]:
    """Lê as últimas N linhas do log"""
    try:
        with open(path, "r") as f:
            return list(deque(f, maxlen=lines))
    except:
        return []


def read_full_log(path: str) -> str:
    """Lê o log completo"""
    try:
        return Path(path).read_text()
    except:
        return ""


def extract_last_error(log_content: str) -> str:
    """Extrai o último bloco de erro do log (Go panic/error)"""
    lines = log_content.split("\n")
    error_lines = []
    in_error = False

    for line in lines:
        line_lower = line.lower()
        line_stripped = line.strip()

        # Go stack trace markers
        if line_stripped.startswith("goroutine ") or line_stripped.startswith("runtime."):
            if in_error:
                error_lines.append(line)
            continue

        # Detecta início de erro
        if "error" in line_lower or "panic" in line_lower or "fail" in line_lower:
            in_error = True
            error_lines = [line]  # Reinicia só para NOVO erro
        elif in_error:
            # Continua capturando linhas do stack trace (Go style)
            if (line_stripped.startswith("/") or
                line_stripped.startswith("main.") or
                line_stripped.startswith("runtime.") or
                line_stripped == "" or
                "\t" in line):
                error_lines.append(line)
            else:
                # Fim do stack trace
                in_error = False

    return "\n".join(error_lines) if error_lines else ""


def copy_to_clipboard(text: str) -> bool:
    """Copia texto para o clipboard usando xclip"""
    try:
        process = subprocess.Popen(
            ["xclip", "-selection", "clipboard"],
            stdin=subprocess.PIPE,
            stderr=subprocess.DEVNULL
        )
        process.communicate(text.encode())
        return process.returncode == 0
    except:
        return False


def save_error_to_file(text: str) -> str:
    """Salva erro em arquivo e retorna o path"""
    try:
        Path(ERROR_FILE).write_text(text)
        return ERROR_FILE
    except:
        return ""


def colorize_log_line(line: str) -> Text:
    """Aplica cores às linhas de log"""
    text = Text(line.rstrip())
    line_lower = line.lower()

    if "error" in line_lower or "fail" in line_lower or "panic" in line_lower:
        text.stylize("bold red")
    elif "warn" in line_lower:
        text.stylize("yellow")
    elif "info" in line_lower:
        text.stylize("dim")
    elif "building" in line_lower or "compiled" in line_lower:
        text.stylize("green")
    elif "watching" in line_lower or "ready" in line_lower:
        text.stylize("cyan")
    elif "hmr" in line_lower or "hot" in line_lower:
        text.stylize("magenta")

    return text


def check_for_errors(lines: list[str]) -> bool:
    """Verifica se há erros nas linhas de log"""
    for line in lines:
        line_lower = line.lower()
        if "error" in line_lower or "panic" in line_lower or "fail" in line_lower:
            return True
    return False


def make_services_table() -> Table:
    """Cria tabela de serviços"""
    table = Table(show_header=True, header_style="bold cyan", expand=True)
    table.add_column("Serviço", style="bold")
    table.add_column("CPU%", justify="right", width=8)
    table.add_column("MEM%", justify="right", width=8)
    table.add_column("RAM", justify="right", width=10)
    table.add_column("Status", justify="center", width=8)

    # Wails dev
    wails_stats = get_process_stats(WAILS_PID)
    if wails_stats:
        table.add_row(
            "[green]wails3 dev[/]",
            f"{wails_stats['cpu']:.1f}%",
            f"{wails_stats['mem']:.1f}%",
            f"{wails_stats['rss_mb']:.0f}MB",
            "[green]●[/]",
        )
    else:
        table.add_row("[red]wails3 dev[/]", "-", "-", "-", "[red]○[/]")

    # Go backend (miau-desktop)
    go_pid = find_process_pid("miau-desktop")
    go_stats = get_process_stats(go_pid)
    if go_stats:
        table.add_row(
            "[cyan]Go Backend[/]",
            f"{go_stats['cpu']:.1f}%",
            f"{go_stats['mem']:.1f}%",
            f"{go_stats['rss_mb']:.0f}MB",
            "[green]●[/]",
        )
    else:
        table.add_row("[dim]Go Backend[/]", "-", "-", "-", "[dim]○[/]")

    # Vite (frontend)
    vite_pid = find_process_pid("vite")
    vite_stats = get_process_stats(vite_pid)
    if vite_stats:
        table.add_row(
            "[yellow]Vite (Svelte)[/]",
            f"{vite_stats['cpu']:.1f}%",
            f"{vite_stats['mem']:.1f}%",
            f"{vite_stats['rss_mb']:.0f}MB",
            "[green]●[/]",
        )
    else:
        table.add_row("[dim]Vite (Svelte)[/]", "-", "-", "-", "[dim]○[/]")

    # SQLite database
    db_stats = get_db_stats()
    if db_stats["exists"]:
        table.add_row(
            "[magenta]SQLite DB[/]",
            "-",
            "-",
            f"{db_stats['size_mb']:.1f}MB",
            "[green]●[/]",
        )
    else:
        table.add_row("[dim]SQLite DB[/]", "-", "-", "-", "[dim]○[/]")

    return table


def make_layout() -> Layout:
    """Cria o layout da TUI"""
    layout = Layout()
    layout.split_column(
        Layout(name="header", size=3),
        Layout(name="main"),
        Layout(name="footer", size=3),
    )
    layout["main"].split_column(
        Layout(name="services", size=10),
        Layout(name="logs"),
    )
    return layout


def generate_display(layout: Layout) -> Layout:
    """Gera o display completo"""
    global has_error

    # Header
    header = Text()
    header.append("  miau ", style="bold cyan")
    header.append("DEV MONITOR", style="bold green")
    header.append("  │  ", style="dim")
    header.append("http://localhost:9245", style="bold green underline")
    header.append("  │  ", style="dim")
    header.append("Ctrl+C para parar", style="yellow")
    layout["header"].update(Panel(header, style="cyan"))

    # Services table
    layout["services"].update(
        Panel(make_services_table(), title="[bold]Serviços[/]", border_style="cyan")
    )

    # Logs
    log_lines = read_log_tail(LOG_FILE)
    has_error = check_for_errors(log_lines)

    if log_lines:
        log_text = Text()
        for line in log_lines:
            log_text.append_text(colorize_log_line(line))
            log_text.append("\n")

        title = "[bold]Logs (Wails)[/]"
        if has_error:
            title += " [red][E] para copiar erro[/]"

        layout["logs"].update(
            Panel(log_text, title=title, border_style="red" if has_error else "green")
        )
    else:
        layout["logs"].update(
            Panel(
                "[dim]Aguardando logs...[/]",
                title="[bold]Logs (Wails)[/]",
                border_style="dim",
            )
        )

    # Footer - system stats + status
    sys_stats = get_system_stats()
    footer = Text()

    # Status message (se houver)
    if last_status_message and (time.time() - last_status_time) < 5:
        footer.append(last_status_message, style="bold green")
        footer.append("  │  ", style="dim")

    footer.append(f"CPU {sys_stats['cpu']:.1f}%", style="cyan")
    footer.append("  │  ", style="dim")
    footer.append(
        f"RAM {sys_stats['mem_used']}MB / {sys_stats['mem_total']}MB ({sys_stats['mem_pct']:.1f}%)",
        style="cyan",
    )
    footer.append("  │  ", style="dim")
    footer.append(f"Atualiza: {REFRESH_INTERVAL}s", style="dim")
    layout["footer"].update(Panel(footer, style="dim"))

    return layout


def handle_keypress(key: str) -> None:
    """Processa tecla pressionada"""
    global last_status_message, last_status_time

    if key.lower() == "e":
        # Copiar erro do log
        log_content = read_full_log(LOG_FILE)
        error = extract_last_error(log_content)
        if error:
            if copy_to_clipboard(error):
                last_status_message = "Erro copiado!"
            else:
                path = save_error_to_file(error)
                last_status_message = f"Erro salvo em {path}"
            last_status_time = time.time()
        else:
            last_status_message = "Nenhum erro encontrado"
            last_status_time = time.time()


def main():
    console = Console()
    layout = make_layout()

    # Salva configuração do terminal
    old_settings = termios.tcgetattr(sys.stdin)

    try:
        # Coloca terminal em modo raw para capturar teclas
        tty.setcbreak(sys.stdin.fileno())

        with Live(layout, console=console, refresh_per_second=1, screen=True) as live:
            while True:
                # Verifica se há tecla pressionada (non-blocking)
                if select.select([sys.stdin], [], [], 0)[0]:
                    key = sys.stdin.read(1)
                    handle_keypress(key)

                live.update(generate_display(layout))
                time.sleep(REFRESH_INTERVAL)
    except KeyboardInterrupt:
        pass
    finally:
        # Restaura configuração do terminal
        termios.tcsetattr(sys.stdin, termios.TCSADRAIN, old_settings)


if __name__ == "__main__":
    main()
