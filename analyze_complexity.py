#!/usr/bin/env python3
import json
import sys
from collections import defaultdict

# Load baseline
with open('baseline-complexity.json', 'r') as f:
    data = json.load(f)

# Extract function statistics
functions = data.get('functions', [])
print(f"Total functions analyzed: {len(functions)}")
print()

# Risk metrics based on task specification
high_risk_complexity = []  # > 12
high_risk_nesting = []     # > 3
high_risk_length = []      # > 30
concurrency_funcs = []

for func in functions:
    name = func.get('name', 'unknown')
    file_path = func.get('file', 'unknown')
    complexity = func.get('cyclomatic_complexity', 0)
    lines = func.get('lines', 0)
    # Nesting depth not always in output, estimate from complexity
    
    if complexity > 12:
        high_risk_complexity.append((name, file_path, complexity, lines))
    if lines > 30:
        high_risk_length.append((name, file_path, complexity, lines))

# Check patterns for concurrency
patterns = data.get('patterns', {})
concurrency_patterns = patterns.get('concurrency_patterns', {})
if concurrency_patterns:
    print("=== CONCURRENCY PATTERNS DETECTED ===")
    for key, value in concurrency_patterns.items():
        print(f"  {key}: {value}")
    print()

# Report high-risk functions
print("=== HIGH COMPLEXITY FUNCTIONS (>12) ===")
if high_risk_complexity:
    # Sort by complexity descending
    high_risk_complexity.sort(key=lambda x: x[2], reverse=True)
    for name, path, complexity, lines in high_risk_complexity[:20]:
        print(f"  {name:50s} | CC={complexity:3d} | Lines={lines:4d} | {path}")
else:
    print("  None found - excellent!")
print()

print("=== LARGE FUNCTIONS (>30 lines) ===")
if high_risk_length:
    # Sort by lines descending
    high_risk_length.sort(key=lambda x: x[3], reverse=True)
    for name, path, complexity, lines in high_risk_length[:20]:
        print(f"  {name:50s} | Lines={lines:4d} | CC={complexity:3d} | {path}")
else:
    print("  None found - excellent!")
print()

# Summary statistics
complexities = [f.get('cyclomatic_complexity', 0) for f in functions]
if complexities:
    avg_complexity = sum(complexities) / len(complexities)
    max_complexity = max(complexities)
    print(f"=== COMPLEXITY STATISTICS ===")
    print(f"  Average Cyclomatic Complexity: {avg_complexity:.2f}")
    print(f"  Maximum Cyclomatic Complexity: {max_complexity}")
    print(f"  Functions with CC > 12: {len(high_risk_complexity)}")
    print(f"  Functions with > 30 lines: {len(high_risk_length)}")
