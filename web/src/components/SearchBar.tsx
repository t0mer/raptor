import { useEffect, useState } from 'react'

interface Props {
  onSearch: (q: string) => void
}

// SearchBar debounces input and emits the request search-DSL query upstream.
export function SearchBar({ onSearch }: Props) {
  const [value, setValue] = useState('')

  useEffect(() => {
    const id = setTimeout(() => onSearch(value.trim()), 300)
    return () => clearTimeout(id)
  }, [value, onSearch])

  return (
    <div className="p-2 border-b border-border">
      <input
        value={value}
        onChange={(e) => setValue(e.target.value)}
        placeholder="Search:  method:POST  content:charge  …"
        className="w-full rounded-lg border border-border bg-surface-2 px-3 py-1.5 text-sm focus:outline-none focus:ring-2 focus:ring-accent/40"
        aria-label="Search requests"
      />
    </div>
  )
}
