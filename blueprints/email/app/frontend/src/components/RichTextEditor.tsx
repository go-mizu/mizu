import { useEditor, EditorContent } from '@tiptap/react';
import StarterKit from '@tiptap/starter-kit';
import Underline from '@tiptap/extension-underline';
import Link from '@tiptap/extension-link';
import TextAlign from '@tiptap/extension-text-align';
import { TextStyle } from '@tiptap/extension-text-style';
import Color from '@tiptap/extension-color';
import Placeholder from '@tiptap/extension-placeholder';
import { Bold, Italic, Underline as UnderlineIcon, Strikethrough, List, ListOrdered, Link as LinkIcon, AlignLeft, AlignCenter, AlignRight, Undo, Redo } from 'lucide-react';
import { useEffect } from 'react';

interface Props {
  content: string;
  onChange: (html: string) => void;
  placeholder?: string;
  autoFocus?: boolean;
}

export default function RichTextEditor({ content, onChange, placeholder, autoFocus }: Props) {
  const editor = useEditor({
    extensions: [
      StarterKit,
      Underline,
      Link.configure({ openOnClick: false }),
      TextAlign.configure({ types: ['heading', 'paragraph'] }),
      TextStyle,
      Color,
      Placeholder.configure({ placeholder: placeholder || 'Compose your email...' }),
    ],
    content,
    onUpdate: ({ editor }) => {
      onChange(editor.getHTML());
    },
    autofocus: autoFocus ? 'end' : false,
    editorProps: {
      attributes: {
        class: 'prose prose-sm max-w-none focus:outline-none min-h-[200px] px-4 py-2',
      },
    },
  });

  useEffect(() => {
    if (editor && content !== editor.getHTML()) {
      editor.commands.setContent(content, { emitUpdate: false });
    }
  }, [content, editor]);

  if (!editor) return null;

  const ToolBtn = ({ active, onClick, children, title }: { active?: boolean; onClick: () => void; children: React.ReactNode; title: string }) => (
    <button
      type="button"
      onClick={onClick}
      title={title}
      className={`p-1.5 rounded hover:bg-gray-100 ${active ? 'bg-gray-200 text-blue-600' : 'text-gray-600'}`}
    >
      {children}
    </button>
  );

  return (
    <div className="border-t border-gray-200">
      <div className="flex items-center gap-0.5 px-2 py-1 border-b border-gray-100 bg-gray-50/50">
        <ToolBtn active={editor.isActive('bold')} onClick={() => editor.chain().focus().toggleBold().run()} title="Bold">
          <Bold size={16} />
        </ToolBtn>
        <ToolBtn active={editor.isActive('italic')} onClick={() => editor.chain().focus().toggleItalic().run()} title="Italic">
          <Italic size={16} />
        </ToolBtn>
        <ToolBtn active={editor.isActive('underline')} onClick={() => editor.chain().focus().toggleUnderline().run()} title="Underline">
          <UnderlineIcon size={16} />
        </ToolBtn>
        <ToolBtn active={editor.isActive('strike')} onClick={() => editor.chain().focus().toggleStrike().run()} title="Strikethrough">
          <Strikethrough size={16} />
        </ToolBtn>
        <div className="w-px h-5 bg-gray-300 mx-1" />
        <ToolBtn active={editor.isActive('bulletList')} onClick={() => editor.chain().focus().toggleBulletList().run()} title="Bullet List">
          <List size={16} />
        </ToolBtn>
        <ToolBtn active={editor.isActive('orderedList')} onClick={() => editor.chain().focus().toggleOrderedList().run()} title="Numbered List">
          <ListOrdered size={16} />
        </ToolBtn>
        <div className="w-px h-5 bg-gray-300 mx-1" />
        <ToolBtn active={editor.isActive({ textAlign: 'left' })} onClick={() => editor.chain().focus().setTextAlign('left').run()} title="Align Left">
          <AlignLeft size={16} />
        </ToolBtn>
        <ToolBtn active={editor.isActive({ textAlign: 'center' })} onClick={() => editor.chain().focus().setTextAlign('center').run()} title="Align Center">
          <AlignCenter size={16} />
        </ToolBtn>
        <ToolBtn active={editor.isActive({ textAlign: 'right' })} onClick={() => editor.chain().focus().setTextAlign('right').run()} title="Align Right">
          <AlignRight size={16} />
        </ToolBtn>
        <div className="w-px h-5 bg-gray-300 mx-1" />
        <ToolBtn onClick={() => {
          const url = window.prompt('Enter URL');
          if (url) editor.chain().focus().setLink({ href: url }).run();
        }} title="Insert Link" active={editor.isActive('link')}>
          <LinkIcon size={16} />
        </ToolBtn>
        <div className="flex-1" />
        <ToolBtn onClick={() => editor.chain().focus().undo().run()} title="Undo">
          <Undo size={16} />
        </ToolBtn>
        <ToolBtn onClick={() => editor.chain().focus().redo().run()} title="Redo">
          <Redo size={16} />
        </ToolBtn>
      </div>
      <EditorContent editor={editor} />
    </div>
  );
}
