#!/usr/bin/env python3
# /// script
# dependencies = [ "matplotlib", "numpy" ]
# ///

import sys
import csv
import numpy as np
import matplotlib.pyplot as plt
from matplotlib import cm

def main():
    # Read CSV from stdin
    reader = csv.reader(sys.stdin)
    headers = next(reader)

    # Store data by column
    data = {}
    for header in headers:
        data[header] = []

    # Read data rows
    for row in reader:
        for i, value in enumerate(row):
            try:
                data[headers[i]].append(float(value))
            except ValueError:
                # Skip non-numeric values
                continue

    # Set up the plot
    fig, ax = plt.subplots(figsize=(8, 8))

    # Use a colormap for different distributions
    colors = cm.tab10(np.linspace(0, 1, len(headers)))

    # Plot each column as a vertical scatter
    for i, (header, values) in enumerate(data.items()):
        # Create x positions with small random offsets
        x = np.random.normal(i+1, 0.1, size=len(values))
        y = values

        # Plot points with transparency
        ax.scatter(x, y, color=colors[i], alpha=0.6, s=30, edgecolors='none')

        # Add mean and median markers
        mean = np.mean(values)
        median = np.median(values)

        # Mean marker (diamond)
        ax.scatter(i+1, mean, color=colors[i], marker='D', s=80, edgecolor='black', linewidth=0.5)

        # Median marker (line)
        ax.plot([i+0.9, i+1.1], [median, median], color=colors[i], linewidth=2)

    # Customize the plot
    ax.set_xticks(range(1, len(headers)+1))
    ax.set_xticklabels(headers, rotation=45, ha='right')
    ax.set_ylabel('Value')
    ax.set_title('Distribution Comparison (Vertical Scatter)')

    # Add legend for mean/median markers
    from matplotlib.lines import Line2D
    legend_elements = [
        Line2D([0], [0], marker='D', color='w', label='Mean',
               markerfacecolor='black', markersize=10),
        Line2D([0], [0], color='black', label='Median', linewidth=2)
    ]
    ax.legend(handles=legend_elements, loc='upper right')

    ax.set_ylim(0, 2)
    plt.tight_layout()
    plt.show()

if __name__ == "__main__":
    main()
