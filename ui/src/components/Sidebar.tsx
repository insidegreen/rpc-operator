import { Boxes, FolderTree, Workflow } from 'lucide-react'

export type Section = 'pipelines' | 'clusters' | 'projects'

interface Props {
  section: Section
  onSelect: (section: Section) => void
}

const items: Array<{ key: Section; label: string; Icon: typeof Workflow }> = [
  { key: 'pipelines', label: 'Pipelines', Icon: Workflow },
  { key: 'clusters', label: 'Clusters', Icon: Boxes },
  { key: 'projects', label: 'Projects', Icon: FolderTree },
]

export function Sidebar({ section, onSelect }: Props) {
  return (
    <nav style={navStyle}>
      {items.map(({ key, label, Icon }) => {
        const active = section === key
        return (
          <button
            key={key}
            onClick={() => onSelect(key)}
            style={{ ...itemStyle, ...(active ? activeItemStyle : {}) }}
          >
            <Icon size={16} />
            {label}
          </button>
        )
      })}
    </nav>
  )
}

const navStyle: React.CSSProperties = {
  display: 'flex', flexDirection: 'column', gap: 4,
  width: 160, flexShrink: 0, paddingRight: 16,
  borderRight: '1px solid #eee',
}
const itemStyle: React.CSSProperties = {
  display: 'flex', alignItems: 'center', gap: 8,
  padding: '8px 12px', border: 'none', background: 'none',
  borderRadius: 6, cursor: 'pointer', fontSize: 14, color: '#444',
  textAlign: 'left', width: '100%',
}
const activeItemStyle: React.CSSProperties = {
  background: '#eff6ff', color: '#1d4ed8', fontWeight: 600,
}
