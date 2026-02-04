import { useState, useRef, useCallback } from "react";
import { Search, X, SlidersHorizontal } from "lucide-react";
import { useEmailStore } from "../store";

interface SearchFilter {
  key: string;
  label: string;
  operator: string;
}

const FILTERS: SearchFilter[] = [
  { key: "from", label: "From", operator: "from:" },
  { key: "to", label: "To", operator: "to:" },
  { key: "subject", label: "Subject", operator: "subject:" },
  { key: "has_attachment", label: "Has attachment", operator: "has:attachment" },
  { key: "is_unread", label: "Is unread", operator: "is:unread" },
  { key: "is_starred", label: "Is starred", operator: "is:starred" },
  { key: "label", label: "Label", operator: "label:" },
];

export default function SearchBar() {
  const setSearch = useEmailStore((s) => s.setSearch);
  const searchQuery = useEmailStore((s) => s.searchQuery);
  const [value, setValue] = useState(searchQuery);
  const [focused, setFocused] = useState(false);
  const [showFilters, setShowFilters] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);
  const filterRef = useRef<HTMLDivElement>(null);

  const handleSubmit = useCallback(
    (e: React.FormEvent) => {
      e.preventDefault();
      setSearch(value);
      setShowFilters(false);
    },
    [value, setSearch]
  );

  const handleClear = useCallback(() => {
    setValue("");
    setSearch("");
    inputRef.current?.focus();
  }, [setSearch]);

  const handleAddFilter = useCallback(
    (filter: SearchFilter) => {
      const needsValue = filter.key !== "has_attachment" && filter.key !== "is_unread" && filter.key !== "is_starred";
      if (needsValue) {
        setValue((v) => (v ? v + " " : "") + filter.operator);
        setShowFilters(false);
        inputRef.current?.focus();
      } else {
        const newVal = (value ? value + " " : "") + filter.operator;
        setValue(newVal);
        setSearch(newVal);
        setShowFilters(false);
      }
    },
    [value, setSearch]
  );

  const removeChip = useCallback(
    (operator: string) => {
      const newVal = value
        .replace(new RegExp(`\\s*${operator.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")}\\S*`), "")
        .trim();
      setValue(newVal);
      setSearch(newVal);
    },
    [value, setSearch]
  );

  // Extract active filter chips from value
  const activeChips = FILTERS.filter((f) => value.includes(f.operator));

  return (
    <div className="relative" ref={filterRef}>
      <form
        onSubmit={handleSubmit}
        className={`relative flex h-12 items-center rounded-full transition-all duration-200 ${
          focused
            ? "bg-white search-bar-focus-shadow ring-1 ring-gmail-border"
            : "bg-gmail-blue-surface hover:bg-[#dfe6f0] search-bar-shadow"
        }`}
      >
        <button
          type="submit"
          className="flex h-10 w-12 flex-shrink-0 items-center justify-center rounded-full focus-ring"
          aria-label="Search"
        >
          <Search
            className={`h-5 w-5 transition-colors ${
              focused ? "text-gmail-blue" : "text-gmail-text-secondary"
            }`}
          />
        </button>

        {/* Active filter chips */}
        {activeChips.length > 0 && (
          <div className="flex items-center gap-1 pr-1">
            {activeChips.map((chip) => (
              <span
                key={chip.key}
                className="inline-flex items-center gap-1 rounded-full bg-gmail-blue-light px-2 py-0.5 text-xs font-medium text-gmail-blue"
              >
                {chip.label}
                <button
                  type="button"
                  onClick={() => removeChip(chip.operator)}
                  className="hover:text-gmail-blue-hover"
                >
                  <X className="h-3 w-3" />
                </button>
              </span>
            ))}
          </div>
        )}

        <input
          ref={inputRef}
          type="text"
          value={value}
          onChange={(e) => setValue(e.target.value)}
          onFocus={() => setFocused(true)}
          onBlur={() => setTimeout(() => setFocused(false), 200)}
          placeholder="Search mail"
          className="flex-1 bg-transparent text-base text-gmail-text-primary outline-none placeholder:text-gmail-text-secondary"
        />

        {value && (
          <button
            type="button"
            onClick={handleClear}
            className="mr-1 flex h-8 w-8 flex-shrink-0 items-center justify-center rounded-full hover:bg-gmail-surface-variant focus-ring"
            aria-label="Clear search"
          >
            <X className="h-5 w-5 text-gmail-text-secondary" />
          </button>
        )}

        <button
          type="button"
          onClick={() => setShowFilters((v) => !v)}
          className="mr-2 flex h-8 w-8 flex-shrink-0 items-center justify-center rounded-full hover:bg-gmail-surface-variant focus-ring"
          aria-label="Show search options"
        >
          <SlidersHorizontal className="h-4 w-4 text-gmail-text-secondary" />
        </button>
      </form>

      {/* Filter dropdown */}
      {showFilters && (
        <div className="absolute left-0 right-0 top-full z-50 mt-1 rounded-lg border border-gray-200 bg-white p-3 shadow-lg">
          <div className="mb-2 text-xs font-medium text-gray-500">Search filters</div>
          <div className="flex flex-wrap gap-2">
            {FILTERS.map((filter) => (
              <button
                key={filter.key}
                onClick={() => handleAddFilter(filter)}
                className={`rounded-full border px-3 py-1 text-sm transition-colors ${
                  activeChips.some((c) => c.key === filter.key)
                    ? "border-gmail-blue bg-gmail-blue-light text-gmail-blue"
                    : "border-gray-300 text-gray-600 hover:bg-gray-50"
                }`}
              >
                {filter.label}
              </button>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
