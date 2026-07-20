import os
dest = "/home/sandy/source_repos/lariv-in/benchmarks/frappe-benchmark/localhost"
src = "sites/localhost"
if not os.path.exists(dest) and not os.path.islink(dest):
    os.symlink(src, dest)
