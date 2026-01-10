interface KeyboardShortcutsModalProps {
  onClose: () => void;
}

const SHORTCUT_GROUPS = [
  {
    title: 'Navigation',
    shortcuts: [
      { keys: ['Arrow keys'], description: 'Navigate between cells' },
      { keys: ['Tab'], description: 'Move to next cell' },
      { keys: ['Shift', 'Tab'], description: 'Move to previous cell' },
      { keys: ['Cmd/Ctrl', 'Arrow Up'], description: 'Jump to first row' },
      { keys: ['Cmd/Ctrl', 'Arrow Down'], description: 'Jump to last row' },
      { keys: ['Cmd/Ctrl', 'Arrow Left'], description: 'Jump to first column' },
      { keys: ['Cmd/Ctrl', 'Arrow Right'], description: 'Jump to last column' },
    ],
  },
  {
    title: 'Selection',
    shortcuts: [
      { keys: ['Shift', 'Arrow keys'], description: 'Extend cell selection' },
      { keys: ['Cmd/Ctrl', 'A'], description: 'Select all rows' },
      { keys: ['Escape'], description: 'Clear selection' },
    ],
  },
  {
    title: 'Editing',
    shortcuts: [
      { keys: ['Enter'], description: 'Edit selected cell' },
      { keys: ['Escape'], description: 'Cancel editing' },
      { keys: ['Delete / Backspace'], description: 'Clear cell content' },
      { keys: ['Shift', 'Enter'], description: 'Insert new row' },
    ],
  },
  {
    title: 'Clipboard',
    shortcuts: [
      { keys: ['Cmd/Ctrl', 'C'], description: 'Copy selected cells' },
      { keys: ['Cmd/Ctrl', 'X'], description: 'Cut selected cells' },
      { keys: ['Cmd/Ctrl', 'V'], description: 'Paste from clipboard' },
    ],
  },
  {
    title: 'Records',
    shortcuts: [
      { keys: ['Space'], description: 'Expand selected record' },
      { keys: ['Cmd/Ctrl', 'D'], description: 'Duplicate record' },
    ],
  },
  {
    title: 'History',
    shortcuts: [
      { keys: ['Cmd/Ctrl', 'Z'], description: 'Undo' },
      { keys: ['Cmd/Ctrl', 'Shift', 'Z'], description: 'Redo' },
      { keys: ['Cmd/Ctrl', 'Y'], description: 'Redo (alternative)' },
    ],
  },
  {
    title: 'View',
    shortcuts: [
      { keys: ['Cmd/Ctrl', 'F'], description: 'Find in view' },
      { keys: ['?'], description: 'Show keyboard shortcuts' },
    ],
  },
];

function KeyBadge({ children }: { children: string }) {
  return (
    <kbd className="inline-flex items-center justify-center px-2 py-1 text-xs font-mono font-medium text-gray-700 bg-gray-100 border border-gray-300 rounded shadow-sm min-w-[24px]">
      {children}
    </kbd>
  );
}

export function KeyboardShortcutsModal({ onClose }: KeyboardShortcutsModalProps) {
  return (
    <div className="modal-overlay" onClick={onClose}>
      <div
        className="modal-content max-w-2xl max-h-[80vh] overflow-hidden"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="modal-header">
          <h3 className="text-lg font-semibold flex items-center gap-2">
            <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 6v6m0 0v6m0-6h6m-6 0H6" />
            </svg>
            Keyboard Shortcuts
          </h3>
          <button onClick={onClose} className="text-gray-400 hover:text-gray-600">
            <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        <div className="modal-body overflow-auto max-h-[60vh]">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            {SHORTCUT_GROUPS.map((group) => (
              <div key={group.title}>
                <h4 className="text-sm font-semibold text-gray-900 mb-3 pb-1 border-b border-gray-200">
                  {group.title}
                </h4>
                <div className="space-y-2">
                  {group.shortcuts.map((shortcut, idx) => (
                    <div key={idx} className="flex items-center justify-between gap-4">
                      <span className="text-sm text-gray-600">{shortcut.description}</span>
                      <div className="flex items-center gap-1 flex-shrink-0">
                        {shortcut.keys.map((key, keyIdx) => (
                          <span key={keyIdx} className="flex items-center gap-1">
                            <KeyBadge>{key}</KeyBadge>
                            {keyIdx < shortcut.keys.length - 1 && (
                              <span className="text-gray-400 text-xs">+</span>
                            )}
                          </span>
                        ))}
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            ))}
          </div>
        </div>

        <div className="modal-footer">
          <p className="text-xs text-gray-500">
            Press <KeyBadge>?</KeyBadge> anytime to show this dialog
          </p>
          <button onClick={onClose} className="btn btn-primary">
            Close
          </button>
        </div>
      </div>
    </div>
  );
}
