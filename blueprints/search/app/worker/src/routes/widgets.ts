import { Hono } from 'hono'
import { KVStore } from '../store/kv'
import { SuggestService } from '../services/suggest'
import { CacheStore } from '../store/cache'
import type { HonoEnv } from '../types'

interface CheatsheetItem {
  syntax: string
  description: string
}

interface CheatsheetSection {
  title: string
  items: CheatsheetItem[]
}

interface Cheatsheet {
  language: string
  title: string
  sections: CheatsheetSection[]
}

const CHEATSHEETS: Record<string, Cheatsheet> = {
  javascript: {
    language: 'javascript',
    title: 'JavaScript Cheatsheet',
    sections: [
      {
        title: 'Variables & Types',
        items: [
          { syntax: 'const x = 1', description: 'Declare a block-scoped constant' },
          { syntax: 'let y = 2', description: 'Declare a block-scoped variable' },
          { syntax: 'typeof x', description: 'Returns the type of a variable as a string' },
          { syntax: 'Array.isArray(arr)', description: 'Check if a value is an array' },
          { syntax: 'x ?? fallback', description: 'Nullish coalescing - use fallback if x is null/undefined' },
          { syntax: 'x?.prop', description: 'Optional chaining - access property safely' },
        ],
      },
      {
        title: 'Functions',
        items: [
          { syntax: 'function name(a, b) {}', description: 'Function declaration with hoisting' },
          { syntax: 'const fn = (a, b) => a + b', description: 'Arrow function expression' },
          { syntax: 'const fn = (...args) => {}', description: 'Rest parameters collect remaining arguments' },
          { syntax: 'fn(a, ...arr)', description: 'Spread syntax expands array into arguments' },
          { syntax: 'function fn(a = 1) {}', description: 'Default parameter values' },
          { syntax: 'const { a, b } = obj', description: 'Destructuring assignment from objects' },
        ],
      },
      {
        title: 'Arrays',
        items: [
          { syntax: 'arr.map(x => x * 2)', description: 'Create new array by transforming each element' },
          { syntax: 'arr.filter(x => x > 0)', description: 'Create new array with elements passing a test' },
          { syntax: 'arr.reduce((acc, x) => acc + x, 0)', description: 'Reduce array to single value' },
          { syntax: 'arr.find(x => x.id === 1)', description: 'Find first element matching condition' },
          { syntax: 'arr.some(x => x > 5)', description: 'Test if any element passes condition' },
          { syntax: 'arr.flat(depth)', description: 'Flatten nested arrays to specified depth' },
          { syntax: '[...arr1, ...arr2]', description: 'Concatenate arrays using spread' },
        ],
      },
      {
        title: 'Async',
        items: [
          { syntax: 'async function fn() {}', description: 'Declare async function returning a Promise' },
          { syntax: 'const result = await promise', description: 'Wait for promise to resolve' },
          { syntax: 'Promise.all([p1, p2])', description: 'Wait for all promises to resolve' },
          { syntax: 'Promise.race([p1, p2])', description: 'Resolve with the first settled promise' },
          { syntax: 'try { await fn() } catch(e) {}', description: 'Handle async errors with try/catch' },
        ],
      },
    ],
  },
  python: {
    language: 'python',
    title: 'Python Cheatsheet',
    sections: [
      {
        title: 'Variables & Types',
        items: [
          { syntax: 'x: int = 10', description: 'Variable with type annotation' },
          { syntax: 'type(x)', description: 'Get the type of a variable' },
          { syntax: 'isinstance(x, int)', description: 'Check if object is instance of a type' },
          { syntax: 'x = y if cond else z', description: 'Ternary conditional expression' },
          { syntax: 'a, b = 1, 2', description: 'Multiple assignment / tuple unpacking' },
          { syntax: 'f"Hello {name}"', description: 'F-string for string interpolation' },
        ],
      },
      {
        title: 'Collections',
        items: [
          { syntax: '[x*2 for x in lst]', description: 'List comprehension' },
          { syntax: '{k: v for k, v in items}', description: 'Dictionary comprehension' },
          { syntax: 'set(lst)', description: 'Create a set from an iterable' },
          { syntax: 'dict.get(key, default)', description: 'Get value with a default fallback' },
          { syntax: 'lst.sort(key=lambda x: x.name)', description: 'Sort list in-place by key function' },
          { syntax: 'enumerate(lst)', description: 'Iterate with index and value' },
          { syntax: 'zip(lst1, lst2)', description: 'Iterate over multiple lists in parallel' },
        ],
      },
      {
        title: 'Functions & Classes',
        items: [
          { syntax: 'def fn(a: int, b: int = 0) -> int:', description: 'Function with type hints and default' },
          { syntax: 'lambda x: x * 2', description: 'Anonymous function expression' },
          { syntax: '*args, **kwargs', description: 'Variadic positional and keyword arguments' },
          { syntax: '@decorator', description: 'Apply a decorator to a function or class' },
          { syntax: 'class Foo(Bar):', description: 'Class definition with inheritance' },
          { syntax: '@property', description: 'Define a getter property on a class' },
        ],
      },
      {
        title: 'Control Flow',
        items: [
          { syntax: 'for x in range(10):', description: 'Loop over a range of numbers' },
          { syntax: 'while cond:', description: 'Loop while condition is true' },
          { syntax: 'with open(f) as fh:', description: 'Context manager for resource handling' },
          { syntax: 'try: ... except E as e:', description: 'Exception handling with specific type' },
          { syntax: 'match value: case pattern:', description: 'Structural pattern matching (3.10+)' },
        ],
      },
    ],
  },
  go: {
    language: 'go',
    title: 'Go Cheatsheet',
    sections: [
      {
        title: 'Variables & Types',
        items: [
          { syntax: 'var x int = 10', description: 'Explicit variable declaration with type' },
          { syntax: 'x := 10', description: 'Short variable declaration with type inference' },
          { syntax: 'const Pi = 3.14', description: 'Constant declaration' },
          { syntax: 'type Point struct { X, Y int }', description: 'Define a struct type' },
          { syntax: 'type Reader interface { Read([]byte) (int, error) }', description: 'Define an interface' },
          { syntax: 'p := &Point{1, 2}', description: 'Create pointer to struct literal' },
        ],
      },
      {
        title: 'Functions & Methods',
        items: [
          { syntax: 'func add(a, b int) int {}', description: 'Function with parameters and return type' },
          { syntax: 'func div(a, b int) (int, error) {}', description: 'Function with multiple return values' },
          { syntax: 'func (p *Point) Scale(f int) {}', description: 'Method with pointer receiver' },
          { syntax: 'func process(fn func(int) int) {}', description: 'Function as parameter' },
          { syntax: 'defer file.Close()', description: 'Defer execution until function returns' },
          { syntax: 'func variadic(nums ...int) {}', description: 'Variadic function parameters' },
        ],
      },
      {
        title: 'Concurrency',
        items: [
          { syntax: 'go fn()', description: 'Launch a goroutine' },
          { syntax: 'ch := make(chan int)', description: 'Create an unbuffered channel' },
          { syntax: 'ch := make(chan int, 10)', description: 'Create a buffered channel' },
          { syntax: 'select { case v := <-ch: ... }', description: 'Select on multiple channel operations' },
          { syntax: 'var mu sync.Mutex', description: 'Mutex for protecting shared state' },
          { syntax: 'var wg sync.WaitGroup', description: 'Wait for a group of goroutines to finish' },
        ],
      },
      {
        title: 'Collections',
        items: [
          { syntax: 'sl := []int{1, 2, 3}', description: 'Slice literal' },
          { syntax: 'sl = append(sl, 4)', description: 'Append to a slice' },
          { syntax: 'm := map[string]int{"a": 1}', description: 'Map literal' },
          { syntax: 'v, ok := m["key"]', description: 'Map lookup with existence check' },
          { syntax: 'for i, v := range sl {}', description: 'Range over slice with index and value' },
          { syntax: 'copy(dst, src)', description: 'Copy elements between slices' },
        ],
      },
    ],
  },
  rust: {
    language: 'rust',
    title: 'Rust Cheatsheet',
    sections: [
      {
        title: 'Variables & Types',
        items: [
          { syntax: 'let x: i32 = 10;', description: 'Immutable variable binding with type' },
          { syntax: 'let mut y = 20;', description: 'Mutable variable binding' },
          { syntax: 'const MAX: u32 = 100;', description: 'Compile-time constant' },
          { syntax: 'let s = String::from("hello");', description: 'Create an owned String from a literal' },
          { syntax: 'let r: &str = &s;', description: 'Borrow as a string slice reference' },
          { syntax: 'type Point = (f64, f64);', description: 'Type alias for a tuple' },
        ],
      },
      {
        title: 'Ownership & Borrowing',
        items: [
          { syntax: 'let s2 = s1.clone();', description: 'Deep clone to avoid move' },
          { syntax: 'fn borrow(s: &String) {}', description: 'Immutable reference parameter' },
          { syntax: 'fn mutate(s: &mut String) {}', description: 'Mutable reference parameter' },
          { syntax: "let r = &v[0..3];", description: 'Slice reference to a portion of a collection' },
          { syntax: "Box::new(value)", description: 'Heap-allocate a value with Box' },
          { syntax: "Rc::clone(&shared)", description: 'Reference-counted shared ownership' },
        ],
      },
      {
        title: 'Enums & Pattern Matching',
        items: [
          { syntax: 'enum Option<T> { Some(T), None }', description: 'Generic enum for optional values' },
          { syntax: 'match value { Pat => expr }', description: 'Exhaustive pattern matching' },
          { syntax: 'if let Some(v) = opt { }', description: 'Conditional pattern match' },
          { syntax: 'value.unwrap_or(default)', description: 'Unwrap Option/Result with fallback' },
          { syntax: 'result?', description: 'Propagate errors with the ? operator' },
        ],
      },
      {
        title: 'Traits & Generics',
        items: [
          { syntax: 'trait Summary { fn summarize(&self) -> String; }', description: 'Define a trait with a method' },
          { syntax: 'impl Summary for Article {}', description: 'Implement a trait for a type' },
          { syntax: 'fn print<T: Display>(val: T) {}', description: 'Generic function with trait bound' },
          { syntax: 'fn process(item: &dyn Summary) {}', description: 'Dynamic dispatch with trait object' },
          { syntax: '#[derive(Debug, Clone)]', description: 'Auto-derive trait implementations' },
        ],
      },
    ],
  },
  html: {
    language: 'html',
    title: 'HTML Cheatsheet',
    sections: [
      {
        title: 'Document Structure',
        items: [
          { syntax: '<!DOCTYPE html>', description: 'HTML5 document type declaration' },
          { syntax: '<html lang="en">', description: 'Root element with language attribute' },
          { syntax: '<head>', description: 'Container for metadata, links, and scripts' },
          { syntax: '<meta charset="utf-8">', description: 'Character encoding declaration' },
          { syntax: '<meta name="viewport" content="width=device-width">', description: 'Responsive viewport setting' },
          { syntax: '<link rel="stylesheet" href="style.css">', description: 'Link external stylesheet' },
        ],
      },
      {
        title: 'Semantic Elements',
        items: [
          { syntax: '<header>', description: 'Introductory content or navigation container' },
          { syntax: '<nav>', description: 'Navigation links section' },
          { syntax: '<main>', description: 'Dominant content of the document' },
          { syntax: '<article>', description: 'Self-contained composition' },
          { syntax: '<section>', description: 'Thematic grouping of content' },
          { syntax: '<aside>', description: 'Content tangentially related to surrounding content' },
          { syntax: '<footer>', description: 'Footer for nearest section or root' },
        ],
      },
      {
        title: 'Forms',
        items: [
          { syntax: '<form action="/submit" method="post">', description: 'Form with action URL and method' },
          { syntax: '<input type="text" name="q" required>', description: 'Text input with validation' },
          { syntax: '<input type="email" placeholder="you@example.com">', description: 'Email input with placeholder' },
          { syntax: '<select name="option"><option>A</option></select>', description: 'Dropdown select element' },
          { syntax: '<textarea rows="4" cols="50"></textarea>', description: 'Multi-line text input' },
          { syntax: '<button type="submit">Send</button>', description: 'Submit button element' },
        ],
      },
      {
        title: 'Media & Embedding',
        items: [
          { syntax: '<img src="img.png" alt="Description">', description: 'Image with alt text for accessibility' },
          { syntax: '<picture><source srcset="img.webp" type="image/webp"></picture>', description: 'Responsive image with multiple sources' },
          { syntax: '<video src="vid.mp4" controls></video>', description: 'Video element with playback controls' },
          { syntax: '<audio src="sound.mp3" controls></audio>', description: 'Audio element with playback controls' },
          { syntax: '<canvas id="c" width="300" height="200"></canvas>', description: 'Drawing surface for graphics' },
        ],
      },
    ],
  },
  css: {
    language: 'css',
    title: 'CSS Cheatsheet',
    sections: [
      {
        title: 'Selectors',
        items: [
          { syntax: '.class', description: 'Select elements by class name' },
          { syntax: '#id', description: 'Select element by ID' },
          { syntax: 'parent > child', description: 'Direct child combinator' },
          { syntax: 'a:hover', description: 'Pseudo-class for hover state' },
          { syntax: 'p::first-line', description: 'Pseudo-element for first line of text' },
          { syntax: '[data-attr="value"]', description: 'Attribute selector with value match' },
          { syntax: ':has(.child)', description: 'Parent selector based on child (CSS4)' },
        ],
      },
      {
        title: 'Layout',
        items: [
          { syntax: 'display: flex;', description: 'Enable flexbox layout on container' },
          { syntax: 'display: grid;', description: 'Enable grid layout on container' },
          { syntax: 'grid-template-columns: repeat(3, 1fr);', description: 'Three equal-width grid columns' },
          { syntax: 'justify-content: center;', description: 'Center items along main axis' },
          { syntax: 'align-items: center;', description: 'Center items along cross axis' },
          { syntax: 'gap: 1rem;', description: 'Gap between flex or grid items' },
        ],
      },
      {
        title: 'Responsive',
        items: [
          { syntax: '@media (max-width: 768px) {}', description: 'Media query for mobile screens' },
          { syntax: 'clamp(1rem, 2vw, 3rem)', description: 'Fluid value between min and max' },
          { syntax: '@container (min-width: 400px) {}', description: 'Container query for component-level responsiveness' },
          { syntax: 'aspect-ratio: 16 / 9;', description: 'Maintain aspect ratio on element' },
          { syntax: 'min-width: 0;', description: 'Prevent flex/grid children from overflowing' },
        ],
      },
      {
        title: 'Custom Properties & Functions',
        items: [
          { syntax: '--color: #333;', description: 'Define a CSS custom property' },
          { syntax: 'var(--color, fallback)', description: 'Use custom property with fallback' },
          { syntax: 'calc(100% - 2rem)', description: 'Perform calculations in values' },
          { syntax: 'color: oklch(70% 0.15 200);', description: 'OKLCH color space for perceptually uniform colors' },
          { syntax: '@layer base, components;', description: 'Declare cascade layers for specificity control' },
        ],
      },
    ],
  },
  sql: {
    language: 'sql',
    title: 'SQL Cheatsheet',
    sections: [
      {
        title: 'Queries',
        items: [
          { syntax: 'SELECT col FROM table WHERE cond;', description: 'Basic select with condition' },
          { syntax: 'SELECT DISTINCT col FROM table;', description: 'Select unique values only' },
          { syntax: 'SELECT * FROM t ORDER BY col DESC LIMIT 10;', description: 'Sort descending and limit rows' },
          { syntax: 'SELECT * FROM t WHERE col LIKE "%pattern%";', description: 'Pattern matching with wildcards' },
          { syntax: 'SELECT * FROM t WHERE col IN (1, 2, 3);', description: 'Match against a list of values' },
          { syntax: 'SELECT * FROM t WHERE col BETWEEN 1 AND 10;', description: 'Range condition inclusive' },
        ],
      },
      {
        title: 'Joins & Subqueries',
        items: [
          { syntax: 'SELECT * FROM a INNER JOIN b ON a.id = b.a_id;', description: 'Inner join matching rows from both tables' },
          { syntax: 'SELECT * FROM a LEFT JOIN b ON a.id = b.a_id;', description: 'Left join keeps all rows from left table' },
          { syntax: 'SELECT * FROM a CROSS JOIN b;', description: 'Cartesian product of two tables' },
          { syntax: 'SELECT * FROM t WHERE id IN (SELECT id FROM t2);', description: 'Subquery in WHERE clause' },
          { syntax: 'WITH cte AS (SELECT ...) SELECT * FROM cte;', description: 'Common Table Expression (CTE)' },
        ],
      },
      {
        title: 'Aggregation',
        items: [
          { syntax: 'SELECT COUNT(*) FROM table;', description: 'Count total rows' },
          { syntax: 'SELECT col, COUNT(*) FROM t GROUP BY col;', description: 'Group and count per group' },
          { syntax: 'SELECT col, SUM(val) FROM t GROUP BY col HAVING SUM(val) > 100;', description: 'Filter groups with HAVING' },
          { syntax: 'SELECT AVG(col), MIN(col), MAX(col) FROM t;', description: 'Aggregate functions for statistics' },
          { syntax: 'SELECT ROW_NUMBER() OVER (ORDER BY col) FROM t;', description: 'Window function for row numbering' },
        ],
      },
      {
        title: 'Data Modification',
        items: [
          { syntax: 'INSERT INTO t (col1, col2) VALUES (v1, v2);', description: 'Insert a new row' },
          { syntax: 'UPDATE t SET col = val WHERE cond;', description: 'Update existing rows' },
          { syntax: 'DELETE FROM t WHERE cond;', description: 'Delete rows matching condition' },
          { syntax: 'CREATE TABLE t (id INT PRIMARY KEY, name TEXT);', description: 'Create a new table' },
          { syntax: 'ALTER TABLE t ADD COLUMN col TYPE;', description: 'Add a column to existing table' },
        ],
      },
    ],
  },
  bash: {
    language: 'bash',
    title: 'Bash Cheatsheet',
    sections: [
      {
        title: 'Variables & Strings',
        items: [
          { syntax: 'VAR="value"', description: 'Assign a variable (no spaces around =)' },
          { syntax: '${VAR:-default}', description: 'Use default if variable is unset' },
          { syntax: '${#VAR}', description: 'Get length of variable value' },
          { syntax: '${VAR//old/new}', description: 'Replace all occurrences in variable' },
          { syntax: '$(command)', description: 'Command substitution - capture output' },
          { syntax: '"$VAR"', description: 'Double quotes preserve variable expansion' },
        ],
      },
      {
        title: 'Control Flow',
        items: [
          { syntax: 'if [[ $x -gt 0 ]]; then ... fi', description: 'Conditional with numeric comparison' },
          { syntax: 'for f in *.txt; do ... done', description: 'Loop over glob pattern matches' },
          { syntax: 'while read -r line; do ... done < file', description: 'Read file line by line' },
          { syntax: 'case "$VAR" in pat) cmd;; esac', description: 'Pattern matching switch statement' },
          { syntax: '[[ -f file ]] && echo exists', description: 'Test if file exists with short-circuit' },
          { syntax: 'cmd1 || cmd2', description: 'Run cmd2 only if cmd1 fails' },
        ],
      },
      {
        title: 'Pipes & Redirection',
        items: [
          { syntax: 'cmd1 | cmd2', description: 'Pipe stdout of cmd1 to stdin of cmd2' },
          { syntax: 'cmd > file', description: 'Redirect stdout to file (overwrite)' },
          { syntax: 'cmd >> file', description: 'Redirect stdout to file (append)' },
          { syntax: 'cmd 2>&1', description: 'Redirect stderr to stdout' },
          { syntax: 'cmd < file', description: 'Read stdin from file' },
          { syntax: 'cmd1 | tee file | cmd2', description: 'Split output to file and next command' },
        ],
      },
      {
        title: 'Common Commands',
        items: [
          { syntax: 'find . -name "*.log" -mtime +7 -delete', description: 'Find and delete files older than 7 days' },
          { syntax: 'grep -rn "pattern" dir/', description: 'Recursively search for pattern with line numbers' },
          { syntax: 'awk \'{print $1}\' file', description: 'Print first column of each line' },
          { syntax: 'sed -i \'s/old/new/g\' file', description: 'In-place find and replace in file' },
          { syntax: 'xargs -I{} cmd {}', description: 'Execute command for each stdin line' },
        ],
      },
    ],
  },
  git: {
    language: 'git',
    title: 'Git Cheatsheet',
    sections: [
      {
        title: 'Basic Commands',
        items: [
          { syntax: 'git init', description: 'Initialize a new repository' },
          { syntax: 'git clone <url>', description: 'Clone a remote repository' },
          { syntax: 'git add -p', description: 'Interactively stage hunks' },
          { syntax: 'git commit -m "message"', description: 'Commit staged changes with message' },
          { syntax: 'git status', description: 'Show working tree status' },
          { syntax: 'git diff --staged', description: 'Show staged changes vs last commit' },
        ],
      },
      {
        title: 'Branching',
        items: [
          { syntax: 'git branch feature', description: 'Create a new branch' },
          { syntax: 'git checkout -b feature', description: 'Create and switch to new branch' },
          { syntax: 'git switch main', description: 'Switch to an existing branch' },
          { syntax: 'git merge feature', description: 'Merge branch into current branch' },
          { syntax: 'git rebase main', description: 'Rebase current branch onto main' },
          { syntax: 'git branch -d feature', description: 'Delete a merged branch' },
        ],
      },
      {
        title: 'Remote Operations',
        items: [
          { syntax: 'git remote add origin <url>', description: 'Add a remote repository' },
          { syntax: 'git fetch origin', description: 'Download objects and refs from remote' },
          { syntax: 'git pull --rebase', description: 'Fetch and rebase local commits on top' },
          { syntax: 'git push -u origin feature', description: 'Push branch and set upstream tracking' },
          { syntax: 'git push origin --delete feature', description: 'Delete a remote branch' },
        ],
      },
      {
        title: 'History & Undo',
        items: [
          { syntax: 'git log --oneline --graph', description: 'Compact log with branch graph' },
          { syntax: 'git stash', description: 'Stash uncommitted changes' },
          { syntax: 'git stash pop', description: 'Apply and remove most recent stash' },
          { syntax: 'git reset HEAD~1', description: 'Undo last commit, keep changes staged' },
          { syntax: 'git revert <hash>', description: 'Create commit undoing a specific commit' },
          { syntax: 'git cherry-pick <hash>', description: 'Apply a commit from another branch' },
        ],
      },
    ],
  },
  regex: {
    language: 'regex',
    title: 'Regular Expressions Cheatsheet',
    sections: [
      {
        title: 'Character Classes',
        items: [
          { syntax: '.', description: 'Match any character except newline' },
          { syntax: '\\d', description: 'Match any digit (0-9)' },
          { syntax: '\\w', description: 'Match word character (letter, digit, underscore)' },
          { syntax: '\\s', description: 'Match whitespace (space, tab, newline)' },
          { syntax: '[abc]', description: 'Match any character in the set' },
          { syntax: '[^abc]', description: 'Match any character NOT in the set' },
          { syntax: '[a-z]', description: 'Match any character in the range' },
        ],
      },
      {
        title: 'Quantifiers',
        items: [
          { syntax: '*', description: 'Match 0 or more times (greedy)' },
          { syntax: '+', description: 'Match 1 or more times (greedy)' },
          { syntax: '?', description: 'Match 0 or 1 time (optional)' },
          { syntax: '{n}', description: 'Match exactly n times' },
          { syntax: '{n,m}', description: 'Match between n and m times' },
          { syntax: '*?', description: 'Match 0 or more times (lazy/non-greedy)' },
        ],
      },
      {
        title: 'Anchors & Boundaries',
        items: [
          { syntax: '^', description: 'Match start of string (or line with m flag)' },
          { syntax: '$', description: 'Match end of string (or line with m flag)' },
          { syntax: '\\b', description: 'Match word boundary' },
          { syntax: '(?=pattern)', description: 'Positive lookahead assertion' },
          { syntax: '(?!pattern)', description: 'Negative lookahead assertion' },
          { syntax: '(?<=pattern)', description: 'Positive lookbehind assertion' },
        ],
      },
      {
        title: 'Groups & References',
        items: [
          { syntax: '(pattern)', description: 'Capturing group' },
          { syntax: '(?:pattern)', description: 'Non-capturing group' },
          { syntax: '(?<name>pattern)', description: 'Named capturing group' },
          { syntax: '\\1', description: 'Backreference to first capturing group' },
          { syntax: 'a|b', description: 'Alternation - match a or b' },
          { syntax: '(?i)', description: 'Case-insensitive flag' },
        ],
      },
    ],
  },
}

const AVAILABLE_LANGUAGES = Object.keys(CHEATSHEETS)

const DEFAULT_WIDGET_SETTINGS = {
  weather: true,
  calculator: true,
  converter: true,
  dictionary: true,
  cheatsheets: true,
  relatedSearches: true,
  knowledgePanel: true,
}

// Widget settings sub-app (mounted at /api/widgets)
const widgetSettingsApp = new Hono<HonoEnv>()

widgetSettingsApp.get('/', async (c) => {
  const kvStore = new KVStore(c.env.SEARCH_KV)
  const settings = await kvStore.getWidgetSettings()
  return c.json(settings ?? DEFAULT_WIDGET_SETTINGS)
})

widgetSettingsApp.put('/', async (c) => {
  const body = await c.req.json()
  const kvStore = new KVStore(c.env.SEARCH_KV)
  const settings = await kvStore.updateWidgetSettings(body)
  return c.json(settings)
})

// Cheatsheet sub-app (mounted at /api/cheatsheet)
const cheatsheetApp = new Hono<HonoEnv>()

cheatsheetApp.get('/:language', (c) => {
  const language = c.req.param('language').toLowerCase()
  const cheatsheet = CHEATSHEETS[language]

  if (!cheatsheet) {
    return c.json(
      { error: `Cheatsheet not found for language: ${language}`, available: AVAILABLE_LANGUAGES },
      404
    )
  }

  return c.json(cheatsheet)
})

// Cheatsheets list sub-app (mounted at /api/cheatsheets)
const cheatsheetsListApp = new Hono<HonoEnv>()

cheatsheetsListApp.get('/', (c) => {
  const list = AVAILABLE_LANGUAGES.map((lang) => ({
    language: lang,
    title: CHEATSHEETS[lang].title,
  }))
  return c.json(list)
})

// Related searches sub-app (mounted at /api/related)
const relatedApp = new Hono<HonoEnv>()

relatedApp.get('/', async (c) => {
  const q = c.req.query('q') ?? ''
  if (!q) {
    return c.json({ error: 'Missing required parameter: q' }, 400)
  }

  const cache = new CacheStore(c.env.SEARCH_KV)
  const suggestService = new SuggestService(cache)
  const suggestions = await suggestService.suggest(q)
  return c.json({ query: q, related: suggestions })
})

export {
  widgetSettingsApp as widgetsRoutes,
  cheatsheetApp as cheatsheetRoutes,
  cheatsheetsListApp as cheatsheetsListRoutes,
  relatedApp as relatedRoutes,
}

export default widgetSettingsApp
