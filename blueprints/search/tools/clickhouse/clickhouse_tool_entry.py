"""PyInstaller entry point — avoids relative-import issues when run as __main__."""
import sys
import os

# Ensure src/ is on the path when running from PyInstaller bundle
sys.path.insert(0, os.path.join(os.path.dirname(__file__), 'src'))

from clickhouse_tool.cli import app_entry

if __name__ == '__main__':
    app_entry()
