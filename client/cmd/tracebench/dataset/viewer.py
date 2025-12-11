#!/usr/bin/env python3
# /// script
# requires-python = ">=3.13"
# dependencies = [ "numpy", "pandas", "matplotlib", "tk" ]
# ///

import argparse
import pandas as pd
import matplotlib.pyplot as plt
from matplotlib.widgets import CheckButtons, Button
import numpy as np
import matplotlib

# use tkinter gui backend
matplotlib.use("TkAgg")  # install tk

parser = argparse.ArgumentParser()
parser.add_argument(
    "--columns",
    nargs=2,
    type=int,
    default=[0, 60],
    metavar="n",
    help="limit the range of columns to view (default: [0, 60])",
)
parser.add_argument(
    "--time",
    nargs=2,
    type=int,
    default=[0, 3600],
    metavar="s",
    help="time range to render initially (default: [0, 3600])",
)
parser.add_argument(
    "--no-log", action="store_true", help="do not use logarithmic x-scale"
)

args = parser.parse_args()
print(args)

# limit the number of rows to view
start = args.columns[0] + 2
end = args.columns[1] + 3

# load datasets
df_requests = pd.read_csv("./requests_minute.csv.gz")
df_tasklens = pd.read_csv("./function_delay_minute.csv.gz")
time_sec = df_requests.iloc[:, 1].values  # time
requests_data = df_requests.iloc[:, start:end].values  # request rates
tasklens_data = df_tasklens.iloc[:, start:end].values  # function delays
column_names = df_requests.columns[start:end]  # column names for checkboxes


# setup plot surface with wider margin for checkboxes
fig, (ax1, ax2) = plt.subplots(
    nrows=2,
    figsize=(16, 10),
    sharex=True,
    gridspec_kw={"height_ratios": [2, 1]},  # requests plot is taller
)
plt.subplots_adjust(left=0.1, bottom=0.05, right=0.98, top=0.95, hspace=0.1)

# plot all lines initially
lines_requests = []
for i in range(requests_data.shape[1]):
    (line,) = ax1.plot(time_sec, requests_data[:, i], label=column_names[i], alpha=0.7)
    lines_requests.append(line)
ax1.set_title("Request rate")
ax1.set_ylabel("Requests / second")
ax1.grid(True, linestyle="--", alpha=0.5)
ax1.set_yscale("linear" if args.no_log else "log")
ax1.set_xlim(*args.time)
# ax.legend(loc="upper right", bbox_to_anchor=(1, 1))

lines_tasklens = []
for i in range(tasklens_data.shape[1]):
    (line,) = ax2.plot(time_sec, tasklens_data[:, i], label=column_names[i], alpha=0.7)
    lines_tasklens.append(line)
ax2.set_title("Task Delay")
ax2.set_ylabel("runtime in seconds")
ax2.grid(True, linestyle="--", alpha=0.5)
ax2.set_yscale("linear" if args.no_log else "log")
ax2.set_xlim(*args.time)
ax2.set_ylim(None, 300)
ax2.set_xlabel("Time (seconds)")


# add checkboxes to toggle lines
rax = plt.axes([0.01, 0.05, 0.05, 0.9])  # Checkbox position
check = CheckButtons(ax=rax, labels=column_names, actives=[False] * len(column_names))

# hide all initially
for col in column_names:
    idx = np.where(column_names == col)[0][0]
    lines_requests[idx].set_visible(False)
    lines_tasklens[idx].set_visible(False)


# callback for checkbox clicks
def toggle_line(label):
    idx = np.where(column_names == label)[0][0]
    lines_requests[idx].set_visible(not lines_requests[idx].get_visible())
    lines_tasklens[idx].set_visible(not lines_tasklens[idx].get_visible())
    fig.canvas.draw_idle()
    # plt.draw()


check.on_clicked(toggle_line)

plt.show()
