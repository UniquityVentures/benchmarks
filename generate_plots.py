import json
import os
import subprocess

def main():
    json_path = "benchmarks/benchmark_metrics.json"
    if not os.path.exists(json_path):
        print(f"Error: {json_path} not found.")
        return

    with open(json_path, "r") as f:
        data = json.load(f)

    os.makedirs("benchmarks/plots_data", exist_ok=True)
    os.makedirs("benchmarks/plots", exist_ok=True)

    # --- 1. Plot Counter & CRUD ---
    for cat in ["counter", "crud"]:
        cat_data = data.get(cat, [])
        if not cat_data:
            continue
        
        # Group by target
        targets = {}
        for item in cat_data:
            target = item["target"]
            workers = item["workers"]
            stats = item.get("stats", {})
            rps = stats.get("AvgRPS", 0)
            latency_ms = stats.get("AvgLatency", 0) / 1_000_000.0 # Convert nanoseconds to ms
            
            if target not in targets:
                targets[target] = []
            targets[target].append((workers, rps, latency_ms))
        
        # Sort targets by workers
        for target in targets:
            targets[target].sort(key=lambda x: x[0])
            
        target_names = sorted(list(targets.keys()))
        dat_path = f"benchmarks/plots_data/{cat}.dat"
        with open(dat_path, "w") as df:
            # Header
            header = "# Workers " + " ".join([f'"{t}_RPS" "{t}_Lat"' for t in target_names])
            df.write(header + "\n")
            
            # Get all unique worker counts
            all_workers = sorted(list(set(w for t in targets for w, _, _ in targets[t])))
            for w in all_workers:
                line = [str(w)]
                for t in target_names:
                    match = [x for x in targets[t] if x[0] == w]
                    if match:
                        line.append(f"{match[0][1]:.2f}")
                        line.append(f"{match[0][2]:.2f}")
                    else:
                        line.append("0.00")
                        line.append("0.00")
                df.write(" ".join(line) + "\n")

        # Generate Gnuplot scripts for clustered histograms
        for metric, ylabel, idx_offset, title_suffix in [
            ("rps", "Average Requests Per Second (RPS)", 0, "Throughput - RPS (Higher is Better)"),
            ("latency", "Average Latency (ms)", 1, "Latency - ms (Lower is Better)")
        ]:
            gp_script = f"""set terminal pngcairo size 1000,600 enhanced font 'Verdana,10'
set output 'benchmarks/plots/{cat}_{metric}.png'
set title "{cat.capitalize()} Benchmark - {title_suffix}"
set xlabel "Concurrent Workers"
set ylabel "{ylabel}"
set grid y
set style data histograms
set style histogram clustered gap 1.5
set style fill solid 0.7 border -1
set boxwidth 0.9
set key left top
"""
            plot_cmds = []
            for i, t in enumerate(target_names):
                col_idx = 2 + i * 2 + idx_offset
                if i == 0:
                    plot_cmds.append(f'"{dat_path}" using {col_idx}:xtic(1) title "{t}"')
                else:
                    plot_cmds.append(f'"{dat_path}" using {col_idx} title "{t}"')
            
            gp_script += "plot " + ", \\\n     ".join(plot_cmds) + "\n"
            
            gp_path = f"benchmarks/plots_data/{cat}_{metric}.gp"
            with open(gp_path, "w") as gpf:
                gpf.write(gp_script)
                
            subprocess.run(["gnuplot", gp_path], check=True)
            print(f"Generated benchmarks/plots/{cat}_{metric}.png")

    # --- 2. Plot Websocket ---
    ws_data = data.get("websocket", [])
    if ws_data:
        stages = {}
        for item in ws_data:
            stage = item["stage"]
            if stage not in stages:
                stages[stage] = []
            stages[stage].append(item)
            
        for stage, stage_items in stages.items():
            targets = {}
            for item in stage_items:
                target = item["target"]
                workers = item["workers"]
                stats = item.get("stats", {})
                rps = stats.get("AvgRPS", 0)
                latency_ms = stats.get("AvgLatency", 0) / 1_000_000.0
                
                if target not in targets:
                    targets[target] = []
                targets[target].append((workers, rps, latency_ms))
                
            for target in targets:
                targets[target].sort(key=lambda x: x[0])
                
            target_names = sorted(list(targets.keys()))
            dat_path = f"benchmarks/plots_data/ws_{stage}.dat"
            with open(dat_path, "w") as df:
                header = "# Workers " + " ".join([f'"{t}_RPS" "{t}_Lat"' for t in target_names])
                df.write(header + "\n")
                
                all_workers = sorted(list(set(w for t in targets for w, _, _ in targets[t])))
                for w in all_workers:
                    line = [str(w)]
                    for t in target_names:
                        match = [x for x in targets[t] if x[0] == w]
                        if match:
                            line.append(f"{match[0][1]:.2f}")
                            line.append(f"{match[0][2]:.2f}")
                        else:
                            line.append("0.00")
                            line.append("0.00")
                    df.write(" ".join(line) + "\n")
                    
            for metric, ylabel, idx_offset, title_suffix in [
                ("rps", "Average Requests Per Second (RPS)", 0, "Throughput - RPS (Higher is Better)"),
                ("latency", "Average Latency (ms)", 1, "Latency - ms (Lower is Better)")
            ]:
                gp_script = f"""set terminal pngcairo size 1000,600 enhanced font 'Verdana,10'
set output 'benchmarks/plots/ws_{stage}_{metric}.png'
set title "Websocket Benchmark ({stage}) - {title_suffix}"
set xlabel "Concurrent Workers"
set ylabel "{ylabel}"
set grid y
set style data histograms
set style histogram clustered gap 1.5
set style fill solid 0.7 border -1
set boxwidth 0.9
set key left top
"""
                plot_cmds = []
                for i, t in enumerate(target_names):
                    col_idx = 2 + i * 2 + idx_offset
                    if i == 0:
                        plot_cmds.append(f'"{dat_path}" using {col_idx}:xtic(1) title "{t}"')
                    else:
                        plot_cmds.append(f'"{dat_path}" using {col_idx} title "{t}"')
                
                gp_script += "plot " + ", \\\n     ".join(plot_cmds) + "\n"
                
                gp_path = f"benchmarks/plots_data/ws_{stage}_{metric}.gp"
                with open(gp_path, "w") as gpf:
                    gpf.write(gp_script)
                    
                subprocess.run(["gnuplot", gp_path], check=True)
                print(f"Generated benchmarks/plots/ws_{stage}_{metric}.png")

if __name__ == "__main__":
    main()
