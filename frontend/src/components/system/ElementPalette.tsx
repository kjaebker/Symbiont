import { Box, Layers, Droplets, Cpu, Activity, Plug, GitBranch, Circle } from 'lucide-react'
import { useDevices } from '@/hooks/useDevices'
import { useProbes } from '@/hooks/useProbes'
import { cn } from '@/lib/utils'

interface PaletteItem {
  nodeType: string
  label: string
  subtype?: string
  deviceId?: number
  probeName?: string
  icon: React.ComponentType<{ size?: number; className?: string }>
  color: string
}

function dragStart(event: React.DragEvent, item: PaletteItem) {
  event.dataTransfer.setData('application/symbiont-node', JSON.stringify(item))
  event.dataTransfer.effectAllowed = 'move'
}

function PaletteEntry({ item }: { item: PaletteItem }) {
  return (
    <div
      draggable
      onDragStart={(e) => dragStart(e, item)}
      className={cn(
        'flex items-center gap-2 px-2.5 py-2 rounded-xl cursor-grab active:cursor-grabbing',
        'bg-surface-container hover:bg-surface-container-high transition-fluid text-on-surface-dim hover:text-on-surface',
      )}
    >
      <item.icon size={13} className={item.color} />
      <span className="text-xs font-medium truncate">{item.label}</span>
    </div>
  )
}

function PaletteSection({ title, items }: { title: string; items: PaletteItem[] }) {
  if (items.length === 0) return null
  return (
    <div className="mb-4">
      <div className="text-[10px] font-semibold text-on-surface-faint uppercase tracking-widest px-1 mb-1.5">
        {title}
      </div>
      <div className="flex flex-col gap-1">
        {items.map((item) => (
          <PaletteEntry key={`${item.nodeType}-${item.label}`} item={item} />
        ))}
      </div>
    </div>
  )
}

export default function ElementPalette() {
  const { data: devicesData } = useDevices()
  const { data: probesData } = useProbes()

  const containers: PaletteItem[] = [
    { nodeType: 'container', label: 'Tank', subtype: 'tank', icon: Box, color: 'text-primary' },
    { nodeType: 'container', label: 'Sump', subtype: 'sump', icon: Layers, color: 'text-primary' },
    { nodeType: 'container', label: 'ATO Reservoir', subtype: 'ato', icon: Droplets, color: 'text-primary' },
  ]

  const devices: PaletteItem[] = (devicesData?.devices ?? []).map((d) => ({
    nodeType: 'device',
    label: d.name,
    deviceId: d.id,
    icon: Cpu,
    color: 'text-secondary',
  }))

  const probes: PaletteItem[] = (probesData?.probes ?? []).map((p) => ({
    nodeType: 'probe',
    label: p.display_name || p.name,
    probeName: p.name,
    icon: Activity,
    color: 'text-secondary',
  }))

  const plumbing: PaletteItem[] = [
    { nodeType: 'tport', label: 'T-Port Junction', icon: GitBranch, color: 'text-primary' },
    { nodeType: 'valve', label: 'Ball Valve', icon: Circle, color: 'text-primary' },
  ]

  return (
    <div className="w-52 shrink-0 bg-surface-container-low p-3 overflow-y-auto flex flex-col">
      <div className="text-xs font-semibold text-on-surface-dim mb-3 px-1 uppercase tracking-widest">
        Elements
      </div>
      <PaletteSection title="Containers" items={containers} />
      <PaletteSection title="Devices" items={devices} />
      <PaletteSection title="Probes" items={probes} />
      <PaletteSection title="Plumbing" items={plumbing} />
    </div>
  )
}
