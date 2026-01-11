import { useBaseStore } from '../../stores/baseStore';
import type { FieldType } from '../../types';

interface GroupBuilderProps {
  onClose: () => void;
}

const GROUPABLE_TYPES: FieldType[] = [
  'single_select', 'multi_select', 'checkbox', 'rating',
  'user', 'created_by', 'last_modified_by',
];

export function GroupBuilder({ onClose }: GroupBuilderProps) {
  const { fields, groupBy, setGroupBy } = useBaseStore();

  const groupableFields = fields.filter(f => GROUPABLE_TYPES.includes(f.type));

  const handleSelectGroup = (fieldId: string | null) => {
    setGroupBy(fieldId);
    onClose();
  };

  return (
    <div className="p-4 min-w-[280px]">
      <div className="flex items-center justify-between mb-4">
        <h3 className="font-semibold text-gray-900">Group records</h3>
        <button onClick={onClose} className="text-gray-400 hover:text-gray-600">
          <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      </div>

      <div className="space-y-1">
        {/* No grouping option */}
        <button
          onClick={() => handleSelectGroup(null)}
          className={`w-full text-left px-3 py-2 rounded-md text-sm flex items-center gap-2 ${
            groupBy === null ? 'bg-primary-50 text-primary' : 'hover:bg-slate-50'
          }`}
        >
          <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6h16M4 10h16M4 14h16M4 18h16" />
          </svg>
          No grouping
        </button>

        {/* Field options */}
        {groupableFields.map((field) => (
          <button
            key={field.id}
            onClick={() => handleSelectGroup(field.id)}
            className={`w-full text-left px-3 py-2 rounded-md text-sm flex items-center gap-2 ${
              groupBy === field.id ? 'bg-primary-50 text-primary' : 'hover:bg-slate-50'
            }`}
          >
            <FieldIcon type={field.type} />
            {field.name}
          </button>
        ))}

        {groupableFields.length === 0 && (
          <p className="text-sm text-gray-500 text-center py-4">
            No fields available for grouping.
            <br />
            Add a Single Select, Checkbox, or Rating field.
          </p>
        )}
      </div>
    </div>
  );
}

function FieldIcon({ type }: { type: FieldType }) {
  switch (type) {
    case 'single_select':
      return (
        <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 9l4-4 4 4m0 6l-4 4-4-4" />
        </svg>
      );
    case 'multi_select':
      return (
        <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2" />
        </svg>
      );
    case 'checkbox':
      return (
        <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
        </svg>
      );
    case 'rating':
      return (
        <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M11.049 2.927c.3-.921 1.603-.921 1.902 0l1.519 4.674a1 1 0 00.95.69h4.915c.969 0 1.371 1.24.588 1.81l-3.976 2.888a1 1 0 00-.363 1.118l1.518 4.674c.3.922-.755 1.688-1.538 1.118l-3.976-2.888a1 1 0 00-1.176 0l-3.976 2.888c-.783.57-1.838-.197-1.538-1.118l1.518-4.674a1 1 0 00-.363-1.118l-3.976-2.888c-.784-.57-.38-1.81.588-1.81h4.914a1 1 0 00.951-.69l1.519-4.674z" />
        </svg>
      );
    case 'user':
    case 'created_by':
    case 'last_modified_by':
      return (
        <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" />
        </svg>
      );
    default:
      return (
        <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6h16M4 10h16M4 14h16M4 18h16" />
        </svg>
      );
  }
}
