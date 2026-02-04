import { useState, useRef, useCallback } from "react";
import { Search, X } from "lucide-react";
import { useEmailStore } from "../store";

export default function SearchBar() {
  const setSearch = useEmailStore((s) => s.setSearch);
  const searchQuery = useEmailStore((s) => s.searchQuery);
  const [value, setValue] = useState(searchQuery);
  const [focused, setFocused] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);

  const handleSubmit = useCallback(
    (e: React.FormEvent) => {
      e.preventDefault();
      setSearch(value);
    },
    [value, setSearch]
  );

  const handleClear = useCallback(() => {
    setValue("");
    setSearch("");
    inputRef.current?.focus();
  }, [setSearch]);

  return (
    <form
      onSubmit={handleSubmit}
      className={`relative flex h-12 items-center rounded-full transition-all ${
        focused
          ? "bg-white search-bar-focus-shadow"
          : "bg-[#EAF1FB] hover:bg-[#dfe6f0] search-bar-shadow"
      }`}
    >
      <button
        type="submit"
        className="flex h-10 w-12 flex-shrink-0 items-center justify-center rounded-full"
        aria-label="Search"
      >
        <Search className="h-5 w-5 text-gmail-text-secondary" />
      </button>
      <input
        ref={inputRef}
        type="text"
        value={value}
        onChange={(e) => setValue(e.target.value)}
        onFocus={() => setFocused(true)}
        onBlur={() => setFocused(false)}
        placeholder="Search mail"
        className="flex-1 bg-transparent text-base outline-none placeholder:text-gmail-text-secondary"
      />
      {value && (
        <button
          type="button"
          onClick={handleClear}
          className="mr-2 flex h-8 w-8 flex-shrink-0 items-center justify-center rounded-full hover:bg-gray-200"
          aria-label="Clear search"
        >
          <X className="h-5 w-5 text-gmail-text-secondary" />
        </button>
      )}
    </form>
  );
}
