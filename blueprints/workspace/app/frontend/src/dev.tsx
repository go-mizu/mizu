/**
 * Development Test Data
 * Comprehensive block examples for testing the block editor
 * Uses API format that gets converted by apiBlocksToBlockNote()
 */

// Helper to create rich text
const text = (t: string, annotations?: Record<string, boolean | string>) => ({
  type: 'text' as const,
  text: t,
  annotations: annotations || {},
})

// All test blocks for the Development Test Page (API format)
export const devTestBlocks = [
  // ========================================
  // Section: Title & Introduction
  // ========================================
  {
    id: 'title-1',
    type: 'heading_1',
    content: { rich_text: [text('Block Editor Showcase')] },
  },
  {
    id: 'intro-1',
    type: 'paragraph',
    content: {
      rich_text: [
        text('This page demonstrates all available block types with various configurations. Use this page to test the editor functionality and verify styling matches Notion.'),
      ],
    },
  },
  { id: 'divider-1', type: 'divider', content: {} },

  // ========================================
  // Section: Text Formatting
  // ========================================
  {
    id: 'h1-formatting',
    type: 'heading_1',
    content: { rich_text: [text('1. Text Formatting')] },
  },
  {
    id: 'text-styles',
    type: 'paragraph',
    content: {
      rich_text: [
        text('Normal text, '),
        text('bold text', { bold: true }),
        text(', '),
        text('italic text', { italic: true }),
        text(', '),
        text('underlined', { underline: true }),
        text(', '),
        text('strikethrough', { strikethrough: true }),
        text(', and '),
        text('inline code', { code: true }),
        text('.'),
      ],
    },
  },
  {
    id: 'text-combined',
    type: 'paragraph',
    content: {
      rich_text: [
        text('Combined: '),
        text('bold italic', { bold: true, italic: true }),
        text(', '),
        text('bold underline', { bold: true, underline: true }),
        text('.'),
      ],
    },
  },
  {
    id: 'text-colors',
    type: 'paragraph',
    content: {
      rich_text: [
        text('Colors: '),
        text('gray ', { color: 'gray' }),
        text('brown ', { color: 'brown' }),
        text('orange ', { color: 'orange' }),
        text('yellow ', { color: 'yellow' }),
        text('green ', { color: 'green' }),
        text('blue ', { color: 'blue' }),
        text('purple ', { color: 'purple' }),
        text('pink ', { color: 'pink' }),
        text('red', { color: 'red' }),
      ],
    },
  },

  // ========================================
  // Section: Headings
  // ========================================
  {
    id: 'h1-headings',
    type: 'heading_1',
    content: { rich_text: [text('2. Headings')] },
  },
  {
    id: 'h1-sample',
    type: 'heading_1',
    content: { rich_text: [text('Heading 1 - Main Title')] },
  },
  {
    id: 'h2-sample',
    type: 'heading_2',
    content: { rich_text: [text('Heading 2 - Section Title')] },
  },
  {
    id: 'h3-sample',
    type: 'heading_3',
    content: { rich_text: [text('Heading 3 - Subsection Title')] },
  },
  {
    id: 'para-compare',
    type: 'paragraph',
    content: { rich_text: [text('Regular paragraph text for comparison.')] },
  },

  // ========================================
  // Section: Lists
  // ========================================
  {
    id: 'h1-lists',
    type: 'heading_1',
    content: { rich_text: [text('3. List Types')] },
  },
  {
    id: 'h2-bullet',
    type: 'heading_2',
    content: { rich_text: [text('Bulleted List')] },
  },
  {
    id: 'bullet-1',
    type: 'bulleted_list_item',
    content: { rich_text: [text('First bullet point')] },
  },
  {
    id: 'bullet-2',
    type: 'bulleted_list_item',
    content: { rich_text: [text('Second bullet point')] },
  },
  {
    id: 'bullet-3',
    type: 'bulleted_list_item',
    content: { rich_text: [text('Third bullet point with longer text to test wrapping behavior')] },
  },
  {
    id: 'h2-numbered',
    type: 'heading_2',
    content: { rich_text: [text('Numbered List')] },
  },
  {
    id: 'numbered-1',
    type: 'numbered_list_item',
    content: { rich_text: [text('First numbered item')] },
  },
  {
    id: 'numbered-2',
    type: 'numbered_list_item',
    content: { rich_text: [text('Second numbered item')] },
  },
  {
    id: 'numbered-3',
    type: 'numbered_list_item',
    content: { rich_text: [text('Third numbered item')] },
  },
  {
    id: 'h2-todo',
    type: 'heading_2',
    content: { rich_text: [text('To-Do List')] },
  },
  {
    id: 'todo-1',
    type: 'to_do',
    content: { rich_text: [text('Unchecked task')], checked: false },
  },
  {
    id: 'todo-2',
    type: 'to_do',
    content: { rich_text: [text('Checked/completed task')], checked: true },
  },
  {
    id: 'todo-3',
    type: 'to_do',
    content: { rich_text: [text('Another pending task')], checked: false },
  },
  {
    id: 'h2-toggle',
    type: 'heading_2',
    content: { rich_text: [text('Toggle List (Nested)')] },
  },
  {
    id: 'toggle-1',
    type: 'toggle',
    content: { rich_text: [text('Getting Started Guide')] },
    children: [
      {
        id: 'toggle-1-child-1',
        type: 'paragraph',
        content: { rich_text: [text('Welcome! This toggle contains helpful information to get you started.')] },
      },
      {
        id: 'toggle-1-child-2',
        type: 'bulleted_list_item',
        content: { rich_text: [text('Step 1: Create a new page')] },
      },
      {
        id: 'toggle-1-child-3',
        type: 'bulleted_list_item',
        content: { rich_text: [text('Step 2: Add some blocks')] },
      },
      {
        id: 'toggle-1-child-4',
        type: 'bulleted_list_item',
        content: { rich_text: [text('Step 3: Share with your team')] },
      },
    ],
  },
  {
    id: 'toggle-2',
    type: 'toggle',
    content: { rich_text: [text('Advanced Features')] },
    children: [
      {
        id: 'toggle-2-child-1',
        type: 'paragraph',
        content: { rich_text: [text('Explore these advanced capabilities:')] },
      },
      {
        id: 'toggle-2-nested-1',
        type: 'toggle',
        content: { rich_text: [text('Keyboard Shortcuts')] },
        children: [
          {
            id: 'toggle-2-nested-1-child-1',
            type: 'paragraph',
            content: { rich_text: [text('Cmd/Ctrl + B', { code: true }), text(' - Bold text')] },
          },
          {
            id: 'toggle-2-nested-1-child-2',
            type: 'paragraph',
            content: { rich_text: [text('Cmd/Ctrl + I', { code: true }), text(' - Italic text')] },
          },
          {
            id: 'toggle-2-nested-1-child-3',
            type: 'paragraph',
            content: { rich_text: [text('Cmd/Ctrl + /', { code: true }), text(' - Open slash menu')] },
          },
        ],
      },
      {
        id: 'toggle-2-nested-2',
        type: 'toggle',
        content: { rich_text: [text('Database Features')] },
        children: [
          {
            id: 'toggle-2-nested-2-child-1',
            type: 'bulleted_list_item',
            content: { rich_text: [text('Create tables, boards, and calendars')] },
          },
          {
            id: 'toggle-2-nested-2-child-2',
            type: 'bulleted_list_item',
            content: { rich_text: [text('Filter and sort your data')] },
          },
          {
            id: 'toggle-2-nested-2-child-3',
            type: 'bulleted_list_item',
            content: { rich_text: [text('Link databases together')] },
          },
        ],
      },
    ],
  },
  {
    id: 'toggle-3',
    type: 'toggle',
    content: { rich_text: [text('Pro Tips')] },
    children: [
      {
        id: 'toggle-3-child-1',
        type: 'callout',
        content: {
          rich_text: [text('Use toggle lists to organize FAQ sections, documentation, or any content that benefits from progressive disclosure.')],
          icon: '💡',
          color: 'blue',
        },
      },
    ],
  },

  // ========================================
  // Section: Callouts
  // ========================================
  {
    id: 'h1-callouts',
    type: 'heading_1',
    content: { rich_text: [text('4. Callout Blocks')] },
  },
  {
    id: 'callout-default',
    type: 'callout',
    content: { rich_text: [text('Default callout - great for tips and information')], icon: '💡', color: 'default' },
  },
  {
    id: 'callout-gray',
    type: 'callout',
    content: { rich_text: [text('Gray callout - subtle and neutral')], icon: '📝', color: 'gray' },
  },
  {
    id: 'callout-brown',
    type: 'callout',
    content: { rich_text: [text('Brown callout - warm and earthy')], icon: '🌰', color: 'brown' },
  },
  {
    id: 'callout-orange',
    type: 'callout',
    content: { rich_text: [text('Orange callout - attention-grabbing')], icon: '🔶', color: 'orange' },
  },
  {
    id: 'callout-yellow',
    type: 'callout',
    content: { rich_text: [text('Yellow callout - warning or important note')], icon: '⚠️', color: 'yellow' },
  },
  {
    id: 'callout-green',
    type: 'callout',
    content: { rich_text: [text('Green callout - success or positive feedback')], icon: '✅', color: 'green' },
  },
  {
    id: 'callout-blue',
    type: 'callout',
    content: { rich_text: [text('Blue callout - informational or tips')], icon: 'ℹ️', color: 'blue' },
  },
  {
    id: 'callout-purple',
    type: 'callout',
    content: { rich_text: [text('Purple callout - creative or ideas')], icon: '💜', color: 'purple' },
  },
  {
    id: 'callout-pink',
    type: 'callout',
    content: { rich_text: [text('Pink callout - playful or feminine')], icon: '🌸', color: 'pink' },
  },
  {
    id: 'callout-red',
    type: 'callout',
    content: { rich_text: [text('Red callout - error or danger')], icon: '❌', color: 'red' },
  },

  // ========================================
  // Section: Quote & Divider
  // ========================================
  {
    id: 'h1-quote',
    type: 'heading_1',
    content: { rich_text: [text('5. Quote & Divider')] },
  },
  {
    id: 'quote-1',
    type: 'quote',
    content: { rich_text: [text('This is a blockquote. Great for highlighting important quotes or references. It can span multiple lines and should maintain proper styling.')] },
  },
  { id: 'divider-2', type: 'divider', content: {} },
  {
    id: 'para-after-divider',
    type: 'paragraph',
    content: { rich_text: [text('Content after the divider.')] },
  },

  // ========================================
  // Section: Code Blocks
  // ========================================
  {
    id: 'h1-code',
    type: 'heading_1',
    content: { rich_text: [text('6. Code Blocks')] },
  },
  {
    id: 'h3-js',
    type: 'heading_3',
    content: { rich_text: [text('JavaScript')] },
  },
  {
    id: 'code-js',
    type: 'code',
    content: {
      language: 'javascript',
      rich_text: [text(`function greeting(name) {
  console.log('Hello, ' + name + '!');
  return {
    message: 'Welcome',
    timestamp: new Date(),
  };
}

// Call the function
greeting('World');`)],
    },
  },
  {
    id: 'h3-python',
    type: 'heading_3',
    content: { rich_text: [text('Python')] },
  },
  {
    id: 'code-python',
    type: 'code',
    content: {
      language: 'python',
      rich_text: [text(`def fibonacci(n):
    """Generate Fibonacci sequence up to n."""
    a, b = 0, 1
    result = []
    while a < n:
        result.append(a)
        a, b = b, a + b
    return result

# Example usage
print(fibonacci(100))`)],
    },
  },
  {
    id: 'h3-go',
    type: 'heading_3',
    content: { rich_text: [text('Go')] },
  },
  {
    id: 'code-go',
    type: 'code',
    content: {
      language: 'go',
      rich_text: [text(`package main

import "fmt"

func main() {
    // Simple hello world
    message := "Hello, Go!"
    fmt.Println(message)

    // Loop example
    for i := 0; i < 5; i++ {
        fmt.Printf("Count: %d\\n", i)
    }
}`)],
    },
  },
  {
    id: 'h3-css',
    type: 'heading_3',
    content: { rich_text: [text('CSS')] },
  },
  {
    id: 'code-css',
    type: 'code',
    content: {
      language: 'css',
      rich_text: [text(`.notion-callout {
  display: flex;
  padding: 16px 16px 16px 12px;
  border-radius: 3px;
  background: rgba(241, 241, 239, 1);
}

.notion-callout:hover {
  background: rgba(235, 235, 233, 1);
}`)],
    },
  },

  // ========================================
  // Section: Equations
  // ========================================
  {
    id: 'h1-equations',
    type: 'heading_1',
    content: { rich_text: [text('7. Equations (LaTeX)')] },
  },
  {
    id: 'para-equations-intro',
    type: 'paragraph',
    content: { rich_text: [text('Mathematical equations rendered with KaTeX. Equations integrate seamlessly with text content.')] },
  },
  {
    id: 'h3-famous',
    type: 'heading_3',
    content: { rich_text: [text('Famous Equations')] },
  },
  {
    id: 'para-einstein',
    type: 'paragraph',
    content: { rich_text: [text("Einstein's mass-energy equivalence:")] },
  },
  {
    id: 'eq-einstein',
    type: 'equation',
    content: { expression: 'E = mc^2' },
  },
  {
    id: 'para-quadratic',
    type: 'paragraph',
    content: { rich_text: [text('The quadratic formula gives the solutions to any quadratic equation:')] },
  },
  {
    id: 'eq-quadratic',
    type: 'equation',
    content: { expression: 'x = \\frac{-b \\pm \\sqrt{b^2 - 4ac}}{2a}' },
  },
  {
    id: 'h3-calculus',
    type: 'heading_3',
    content: { rich_text: [text('Calculus')] },
  },
  {
    id: 'para-gaussian',
    type: 'paragraph',
    content: { rich_text: [text('The Gaussian integral is fundamental in probability theory and statistics:')] },
  },
  {
    id: 'eq-gaussian',
    type: 'equation',
    content: { expression: '\\int_{-\\infty}^{\\infty} e^{-x^2} \\, dx = \\sqrt{\\pi}' },
  },
  {
    id: 'para-derivative',
    type: 'paragraph',
    content: { rich_text: [text('Definition of the derivative:')] },
  },
  {
    id: 'eq-derivative',
    type: 'equation',
    content: { expression: "f'(x) = \\lim_{h \\to 0} \\frac{f(x+h) - f(x)}{h}" },
  },
  {
    id: 'para-taylor',
    type: 'paragraph',
    content: { rich_text: [text('Taylor series expansion around a point:')] },
  },
  {
    id: 'eq-taylor',
    type: 'equation',
    content: { expression: "f(x) = \\sum_{n=0}^{\\infty} \\frac{f^{(n)}(a)}{n!}(x-a)^n" },
  },
  {
    id: 'h3-series',
    type: 'heading_3',
    content: { rich_text: [text('Series & Summations')] },
  },
  {
    id: 'para-basel',
    type: 'paragraph',
    content: { rich_text: [text('The Basel problem, solved by Euler:')] },
  },
  {
    id: 'eq-basel',
    type: 'equation',
    content: { expression: '\\sum_{n=1}^{\\infty} \\frac{1}{n^2} = \\frac{\\pi^2}{6}' },
  },
  {
    id: 'para-euler',
    type: 'paragraph',
    content: { rich_text: [text("Euler's identity, often called the most beautiful equation:")] },
  },
  {
    id: 'eq-euler',
    type: 'equation',
    content: { expression: 'e^{i\\pi} + 1 = 0' },
  },
  {
    id: 'h3-physics',
    type: 'heading_3',
    content: { rich_text: [text('Physics & Engineering')] },
  },
  {
    id: 'para-maxwell',
    type: 'paragraph',
    content: { rich_text: [text("Maxwell's equations in differential form describe electromagnetism:")] },
  },
  {
    id: 'eq-maxwell',
    type: 'equation',
    content: { expression: '\\nabla \\cdot \\mathbf{E} = \\frac{\\rho}{\\varepsilon_0}, \\quad \\nabla \\times \\mathbf{E} = -\\frac{\\partial \\mathbf{B}}{\\partial t}' },
  },
  {
    id: 'para-schrodinger',
    type: 'paragraph',
    content: { rich_text: [text('The time-dependent Schrödinger equation in quantum mechanics:')] },
  },
  {
    id: 'eq-schrodinger',
    type: 'equation',
    content: { expression: 'i\\hbar\\frac{\\partial}{\\partial t}\\Psi(\\mathbf{r},t) = \\hat{H}\\Psi(\\mathbf{r},t)' },
  },
  {
    id: 'h3-matrices',
    type: 'heading_3',
    content: { rich_text: [text('Matrices & Linear Algebra')] },
  },
  {
    id: 'para-matrix',
    type: 'paragraph',
    content: { rich_text: [text('A 2×2 matrix and its determinant:')] },
  },
  {
    id: 'eq-matrix',
    type: 'equation',
    content: { expression: '\\mathbf{A} = \\begin{pmatrix} a & b \\\\ c & d \\end{pmatrix}, \\quad \\det(\\mathbf{A}) = ad - bc' },
  },
  {
    id: 'para-eigen',
    type: 'paragraph',
    content: { rich_text: [text('The eigenvalue equation:')] },
  },
  {
    id: 'eq-eigen',
    type: 'equation',
    content: { expression: '\\mathbf{A}\\mathbf{v} = \\lambda\\mathbf{v}' },
  },

  // ========================================
  // Section: Media Blocks
  // ========================================
  {
    id: 'h1-media',
    type: 'heading_1',
    content: { rich_text: [text('8. Media Blocks')] },
  },
  {
    id: 'h2-image',
    type: 'heading_2',
    content: { rich_text: [text('Image')] },
  },
  {
    id: 'image-1',
    type: 'image',
    content: {
      url: 'https://images.unsplash.com/photo-1506905925346-21bda4d32df4?w=800',
      caption: 'Beautiful mountain landscape',
    },
  },
  {
    id: 'h2-bookmark',
    type: 'heading_2',
    content: { rich_text: [text('Bookmark')] },
  },
  {
    id: 'bookmark-1',
    type: 'bookmark',
    content: {
      url: 'https://github.com',
      title: 'GitHub: Where the world builds software',
      description: 'GitHub is where over 100 million developers shape the future of software.',
    },
  },
  {
    id: 'bookmark-2',
    type: 'bookmark',
    content: {
      url: 'https://notion.so',
      title: 'Notion - The all-in-one workspace',
      description: 'A new tool that blends your everyday work apps into one.',
    },
  },

  // ========================================
  // Section: Advanced Blocks
  // ========================================
  {
    id: 'h1-advanced',
    type: 'heading_1',
    content: { rich_text: [text('9. Advanced Blocks')] },
  },
  {
    id: 'h2-toc',
    type: 'heading_2',
    content: { rich_text: [text('Table of Contents')] },
  },
  {
    id: 'toc-1',
    type: 'table_of_contents',
    content: {},
  },
  {
    id: 'h2-synced',
    type: 'heading_2',
    content: { rich_text: [text('Synced Block')] },
  },
  {
    id: 'synced-1',
    type: 'synced_block',
    content: { sync_id: 'test-sync-id', original_page_name: 'Source Page' },
  },
  {
    id: 'h2-template',
    type: 'heading_2',
    content: { rich_text: [text('Template Button')] },
  },
  {
    id: 'template-1',
    type: 'template_button',
    content: { button_text: 'Add New Task', button_style: 'primary' },
  },
  {
    id: 'h2-breadcrumb',
    type: 'heading_2',
    content: { rich_text: [text('Breadcrumb')] },
  },
  {
    id: 'breadcrumb-1',
    type: 'breadcrumb',
    content: {},
  },

  // ========================================
  // Section: Simple Table
  // ========================================
  {
    id: 'h1-table',
    type: 'heading_1',
    content: { rich_text: [text('10. Simple Table')] },
  },
  {
    id: 'table-1',
    type: 'simple_table',
    content: {
      table_data: JSON.stringify({
        rows: [
          { cells: [{ content: 'Name' }, { content: 'Status' }, { content: 'Priority' }] },
          { cells: [{ content: 'Task 1' }, { content: 'Done' }, { content: 'High' }] },
          { cells: [{ content: 'Task 2' }, { content: 'In Progress' }, { content: 'Medium' }] },
          { cells: [{ content: 'Task 3' }, { content: 'Todo' }, { content: 'Low' }] },
        ],
        hasHeader: true,
      }),
    },
  },

  // ========================================
  // Section: Summary
  // ========================================
  { id: 'divider-final', type: 'divider', content: {} },
  {
    id: 'h1-summary',
    type: 'heading_1',
    content: { rich_text: [text('Summary')] },
  },
  {
    id: 'summary-callout',
    type: 'callout',
    content: {
      rich_text: [
        text('This page includes '),
        text('50+ block types', { bold: true }),
        text(' demonstrating the full capabilities of the block editor.'),
      ],
      icon: '🎯',
      color: 'blue',
    },
  },
  {
    id: 'summary-1',
    type: 'bulleted_list_item',
    content: {
      rich_text: [
        text('Text formatting', { bold: true }),
        text(' - Bold, italic, colors, highlights, inline code'),
      ],
    },
  },
  {
    id: 'summary-2',
    type: 'bulleted_list_item',
    content: {
      rich_text: [
        text('Lists', { bold: true }),
        text(' - Bulleted, numbered, to-do, toggle'),
      ],
    },
  },
  {
    id: 'summary-3',
    type: 'bulleted_list_item',
    content: {
      rich_text: [
        text('Callouts', { bold: true }),
        text(' - All 10 color variants with icons'),
      ],
    },
  },
  {
    id: 'summary-4',
    type: 'bulleted_list_item',
    content: {
      rich_text: [
        text('Code blocks', { bold: true }),
        text(' - JavaScript, Python, Go, CSS with syntax highlighting'),
      ],
    },
  },
  {
    id: 'summary-5',
    type: 'bulleted_list_item',
    content: {
      rich_text: [
        text('Media', { bold: true }),
        text(' - Images, bookmarks'),
      ],
    },
  },
  {
    id: 'summary-6',
    type: 'bulleted_list_item',
    content: {
      rich_text: [
        text('Advanced', { bold: true }),
        text(' - Equations, tables, synced blocks, templates, breadcrumbs'),
      ],
    },
  },
  { id: 'divider-end', type: 'divider', content: {} },
  {
    id: 'footer',
    type: 'paragraph',
    content: {
      rich_text: [
        text('Use this page to test drag-and-drop, formatting, and visual consistency with Notion.', { italic: true, color: 'gray' }),
      ],
    },
  },
]
