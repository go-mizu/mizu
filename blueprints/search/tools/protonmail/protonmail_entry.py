"""PyInstaller entry point."""
import sys
import os

sys.path.insert(0, os.path.join(os.path.dirname(__file__), 'src'))

from protonmail_tool.cli import app_entry

if __name__ == '__main__':
    app_entry()
