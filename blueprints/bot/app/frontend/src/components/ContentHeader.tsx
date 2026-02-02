import { TAB_META } from '../config/navigation';

interface ContentHeaderProps {
  tab: string;
}

export function ContentHeader({ tab }: ContentHeaderProps) {
  const meta = TAB_META[tab] || { title: '', sub: '' };
  return (
    <div className="content-header">
      <div>
        <h1 className="page-title">{meta.title}</h1>
        <p className="page-sub">{meta.sub}</p>
      </div>
    </div>
  );
}
