import json
import os
import subprocess
import tkinter as tk
from tkinter import ttk

# Load JSON metrics
DATA_FILE = "benchmark_metrics.json"

class BenchmarkPlotterGUI:
    def __init__(self, root):
        self.root = root
        self.root.title("Lamu vs Django Benchmark Plotter (Gnuplot)")
        self.root.geometry("900x700")

        # Load metrics data
        if not os.path.exists(DATA_FILE):
            self.show_error(f"Error: {DATA_FILE} not found. Please run the benchmark coordinator first.")
            return

        try:
            with open(DATA_FILE, "r") as f:
                self.data = json.load(f)
        except Exception as e:
            self.show_error(f"Error loading {DATA_FILE}: {e}")
            return

        # Main layout
        control_frame = ttk.LabelFrame(root, text="Controls", padding=10)
        control_frame.pack(side=tk.TOP, fill=tk.X, padx=10, pady=5)

        # Plot display area
        self.plot_frame = ttk.LabelFrame(root, text="Plot Preview", padding=10)
        self.plot_frame.pack(side=tk.TOP, fill=tk.BOTH, expand=True, padx=10, pady=5)

        self.img_label = ttk.Label(self.plot_frame, text="Generate a plot to display preview.")
        self.img_label.pack(fill=tk.BOTH, expand=True)

        # Dropdowns
        ttk.Label(control_frame, text="Type:").grid(row=0, column=0, padx=5, pady=5, sticky=tk.W)
        self.type_var = tk.StringVar()
        self.type_cb = ttk.Combobox(control_frame, textvariable=self.type_var, values=["CRUD", "Counter", "WebSocket"], state="readonly")
        self.type_cb.grid(row=0, column=1, padx=5, pady=5)
        self.type_cb.bind("<<ComboboxSelected>>", self.on_type_change)

        ttk.Label(control_frame, text="Metric:").grid(row=0, column=2, padx=5, pady=5, sticky=tk.W)
        self.metric_var = tk.StringVar()
        self.metric_cb = ttk.Combobox(control_frame, textvariable=self.metric_var, state="readonly")
        self.metric_cb.grid(row=0, column=3, padx=5, pady=5)

        self.workers_label = ttk.Label(control_frame, text="Workers:")
        self.workers_label.grid(row=0, column=4, padx=5, pady=5, sticky=tk.W)
        self.workers_var = tk.StringVar()
        self.workers_cb = ttk.Combobox(control_frame, textvariable=self.workers_var, state="readonly")
        self.workers_cb.grid(row=0, column=5, padx=5, pady=5)

        plot_btn = ttk.Button(control_frame, text="Generate Plot", command=self.generate_plot)
        plot_btn.grid(row=0, column=6, padx=15, pady=5)

        # Initialize dropdown choices
        self.type_cb.current(0)
        self.on_type_change(None)

    def show_error(self, message):
        err_label = ttk.Label(self.root, text=message, foreground="red", font=("Arial", 14))
        err_label.pack(expand=True)

    def on_type_change(self, event):
        btype = self.type_var.get()
        metrics = ["Average RPS", "Max RPS", "Average Latency (ms)", "Max Latency (ms)", "Max Connections", "Avg Connections", "Total Bytes Received (MB)"]
        self.metric_cb["values"] = metrics
        self.metric_cb.current(0)

        if btype == "WebSocket":
            self.workers_cb.grid_remove()
            self.workers_label.grid_remove()
        else:
            self.workers_cb.grid()
            self.workers_label.grid()
            # Extract worker options from data
            key = btype.lower()
            if key in self.data:
                workers = sorted(list(set(item["workers"] for item in self.data[key])))
                self.workers_cb["values"] = workers
                if workers:
                    self.workers_cb.current(0)

    def generate_plot(self):
        btype = self.type_var.get()
        metric = self.metric_var.get()
        key = btype.lower()

        # Temporary files
        data_file = "temp_plot_data.dat"
        script_file = "temp_plot_script.gp"
        png_file = "temp_plot.png"

        # Map metric selections to JSON fields
        field_map = {
            "Average RPS": lambda s: s["avg_rps"],
            "Max RPS": lambda s: s["max_rps"],
            "Average Latency (ms)": lambda s: s["avg_latency"] / 1000000.0, # ns to ms
            "Max Latency (ms)": lambda s: s["max_latency"] / 1000000.0, # ns to ms
            "Max Connections": lambda s: s["max_connections"],
            "Avg Connections": lambda s: s["avg_connections"],
            "Total Bytes Received (MB)": lambda s: s["total_bytes_received"] / (1024.0 * 1024.0)
        }

        val_func = field_map[metric]

        if btype in ["CRUD", "Counter"]:
            if not self.workers_var.get():
                return
            workers = int(self.workers_var.get())
            # Find data matching workers
            items = [item for item in self.data[key] if item["workers"] == workers]
            if not items:
                return

            with open(data_file, "w") as f:
                f.write("Target Value\n")
                for item in items:
                    name = item["target"].replace(" ", "_")
                    val = val_func(item["stats"])
                    f.write(f"{name} {val:.2f}\n")

            ylabel = metric
            title = f"{btype} {metric} (Workers: {workers})"
            gp_script = f"""
set terminal pngcairo size 800,500 enhanced font 'Arial,10'
set output '{png_file}'
set style data histograms
set style fill solid 1.0 border -1
set boxwidth 0.5
set grid y
set ylabel '{ylabel}'
set title '{title}'
plot '{data_file}' using 2:xtic(1) notitle linecolor rgb '#4F46E5'
"""
        else:
            # WebSocket: group by target, X-axis is stages
            ws_stages = [
                ("small", "small"), ("small", "medium"), ("small", "large"),
                ("medium", "small"), ("medium", "medium"), ("medium", "large"),
                ("large", "small"), ("large", "medium"), ("large", "large")
            ]

            # Find all unique targets
            targets = sorted(list(set(item["target"] for item in self.data["websocket"])))
            
            # Map of stageName -> targetName -> value
            stage_map = {}
            for item in self.data["websocket"]:
                stage = item["stage"]
                target = item["target"]
                val = val_func(item["stats"])
                if stage not in stage_map:
                    stage_map[stage] = {}
                stage_map[stage][target] = val

            with open(data_file, "w") as f:
                header = "Stage " + " ".join(t.replace(" ", "_") for t in targets) + "\n"
                f.write(header)
                for client, server in ws_stages:
                    stage_key = f"WS_{client}_req_{server}_resp"
                    row = [f"{client[0]}/{server[0]}"]
                    for t in targets:
                        val = stage_map.get(stage_key, {}).get(t, 0.0)
                        row.append(f"{val:.2f}")
                    f.write(" ".join(row) + "\n")

            ylabel = metric
            title = f"WebSocket {metric}"

            # Plot columns
            plot_cmds = []
            colors = ['#4F46E5', '#0D9488', '#DB2777', '#D97706']
            for idx, t in enumerate(targets):
                col = idx + 2
                name = t.replace("_", " ")
                color = colors[idx % len(colors)]
                xtic = ":xtic(1)" if idx == 0 else ""
                plot_cmds.append(f"'{data_file}' using {col}{xtic} title '{name}' linecolor rgb '{color}'")

            gp_script = f"""
set terminal pngcairo size 800,500 enhanced font 'Arial,10'
set output '{png_file}'
set style data histograms
set style fill solid 1.0 border -1
set style histogram clustered gap 1
set boxwidth 0.9
set grid y
set ylabel '{ylabel}'
set title '{title}'
plot {", ".join(plot_cmds)}
"""

        # Write and execute gnuplot script
        with open(script_file, "w") as f:
            f.write(gp_script)

        try:
            subprocess.run(["gnuplot", script_file], check=True)
            # Display PNG in Tkinter (Python 3.4+ native PNG support in tk.PhotoImage)
            self.tk_img = tk.PhotoImage(file=png_file)
            self.img_label.config(image=self.tk_img, text="")
        except Exception as e:
            self.img_label.config(text=f"Failed to generate plot: {e}\nEnsure 'gnuplot' is installed on your system.")

        # Clean up temp files
        for f in [data_file, script_file, png_file]:
            if os.path.exists(f):
                try:
                    os.remove(f)
                except:
                    pass

if __name__ == "__main__":
    root = tk.Tk()
    app = BenchmarkPlotterGUI(root)
    root.mainloop()
