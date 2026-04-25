import type { PageKey } from '@/lib/pageTheme'

interface SidebarSilhouetteProps {
  page: PageKey
}

function DashboardSilhouette() {
  return (
    <svg viewBox="0 0 220 140" fill="none" xmlns="http://www.w3.org/2000/svg" className="w-full h-full">
      {/* Kelp frond left */}
      <g className="animate-sway" style={{ animationDelay: '0s' }}>
        <path d="M18 140 C16 120 14 100 18 80 C22 60 16 45 20 30" stroke="#3adffa" strokeWidth="2.5" strokeLinecap="round" opacity="0.3"/>
        <path d="M18 100 C10 95 6 85 12 78" stroke="#3adffa" strokeWidth="1.5" strokeLinecap="round" opacity="0.25"/>
        <path d="M18 78 C26 73 30 63 24 56" stroke="#3adffa" strokeWidth="1.5" strokeLinecap="round" opacity="0.2"/>
      </g>
      {/* Staghorn coral center-left */}
      <g className="animate-sway2" style={{ animationDelay: '0.6s' }}>
        <path d="M55 140 L55 100" stroke="#6dfe9c" strokeWidth="3" strokeLinecap="round" opacity="0.35"/>
        <path d="M55 118 L42 100 L38 88" stroke="#6dfe9c" strokeWidth="2.5" strokeLinecap="round" opacity="0.3"/>
        <path d="M55 118 L68 102 L72 90" stroke="#6dfe9c" strokeWidth="2.5" strokeLinecap="round" opacity="0.3"/>
        <path d="M42 100 L34 88 L30 78" stroke="#6dfe9c" strokeWidth="2" strokeLinecap="round" opacity="0.25"/>
        <path d="M68 102 L76 90 L80 80" stroke="#6dfe9c" strokeWidth="2" strokeLinecap="round" opacity="0.25"/>
        <path d="M55 108 L50 95" stroke="#6dfe9c" strokeWidth="1.5" strokeLinecap="round" opacity="0.2"/>
      </g>
      {/* Sea fan center */}
      <g className="animate-sway" style={{ animationDelay: '1.2s' }}>
        <path d="M110 140 L110 60" stroke="#3adffa" strokeWidth="2" strokeLinecap="round" opacity="0.25"/>
        <ellipse cx="110" cy="90" rx="28" ry="42" stroke="#3adffa" strokeWidth="1.5" opacity="0.2" fill="none"/>
        <path d="M110 90 C96 80 90 68 98 60" stroke="#3adffa" strokeWidth="1.2" strokeLinecap="round" opacity="0.18"/>
        <path d="M110 90 C124 80 130 68 122 60" stroke="#3adffa" strokeWidth="1.2" strokeLinecap="round" opacity="0.18"/>
        <path d="M110 110 C96 102 88 90 96 82" stroke="#3adffa" strokeWidth="1.2" strokeLinecap="round" opacity="0.18"/>
        <path d="M110 110 C124 102 132 90 124 82" stroke="#3adffa" strokeWidth="1.2" strokeLinecap="round" opacity="0.18"/>
      </g>
      {/* Brain coral right of center */}
      <g>
        <ellipse cx="155" cy="125" rx="22" ry="16" fill="#6dfe9c" opacity="0.12"/>
        <path d="M133 125 C136 118 142 115 148 118 C152 114 158 114 162 118 C168 115 174 118 177 125" stroke="#6dfe9c" strokeWidth="1.5" opacity="0.25" fill="none"/>
        <path d="M136 128 C140 122 146 120 152 123 C156 119 162 120 166 123 C170 120 175 122 176 128" stroke="#6dfe9c" strokeWidth="1.2" opacity="0.2" fill="none"/>
      </g>
      {/* Anemone tentacles far right */}
      <g className="animate-sway2" style={{ animationDelay: '2s' }}>
        <path d="M200 140 C198 130 204 122 200 115" stroke="#ff8796" strokeWidth="1.5" strokeLinecap="round" opacity="0.2"/>
        <path d="M206 140 C208 128 202 120 208 112" stroke="#ff8796" strokeWidth="1.5" strokeLinecap="round" opacity="0.2"/>
        <path d="M212 140 C214 130 210 122 215 114" stroke="#ff8796" strokeWidth="1.5" strokeLinecap="round" opacity="0.18"/>
        <circle cx="200" cy="114" r="3" fill="#ff8796" opacity="0.25"/>
        <circle cx="208" cy="111" r="3" fill="#ff8796" opacity="0.22"/>
        <circle cx="215" cy="113" r="2.5" fill="#ff8796" opacity="0.2"/>
      </g>
      {/* Tube worms left area */}
      <g className="animate-sway" style={{ animationDelay: '3s' }}>
        <path d="M34 140 C34 132 32 128 34 122" stroke="#6dfe9c" strokeWidth="1.8" strokeLinecap="round" opacity="0.22"/>
        <path d="M39 140 C39 130 41 126 39 119" stroke="#6dfe9c" strokeWidth="1.8" strokeLinecap="round" opacity="0.2"/>
        <circle cx="34" cy="121" r="2.5" fill="#6dfe9c" opacity="0.28"/>
        <circle cx="39" cy="118" r="2.5" fill="#6dfe9c" opacity="0.25"/>
      </g>
      {/* Sandy base */}
      <path d="M0 138 Q55 134 110 137 Q165 140 220 136 L220 140 L0 140 Z" fill="#3adffa" opacity="0.06"/>
    </svg>
  )
}

function LivestockSilhouette() {
  return (
    <svg viewBox="0 0 220 140" fill="none" xmlns="http://www.w3.org/2000/svg" className="w-full h-full">
      {/* Clownfish */}
      <g className="animate-sway" style={{ animationDelay: '0.5s' }}>
        <ellipse cx="60" cy="75" rx="22" ry="12" fill="#ff8796" opacity="0.25"/>
        <path d="M82 75 L94 68 L90 80 Z" fill="#ff8796" opacity="0.22"/>
        <path d="M58 65 C64 60 70 62 72 68" stroke="#3adffa" strokeWidth="1.5" strokeLinecap="round" opacity="0.3" fill="none"/>
        <path d="M58 85 C64 90 70 88 72 82" stroke="#3adffa" strokeWidth="1.5" strokeLinecap="round" opacity="0.3" fill="none"/>
        <circle cx="50" cy="74" r="2" fill="#070d1f" opacity="0.5"/>
      </g>
      {/* Tang */}
      <g className="animate-sway2" style={{ animationDelay: '1.2s' }}>
        <ellipse cx="160" cy="55" rx="26" ry="13" fill="#3adffa" opacity="0.2"/>
        <path d="M186 55 L200 46 L196 62 Z" fill="#3adffa" opacity="0.18"/>
        <path d="M168 44 C174 40 180 42 182 49" stroke="#6dfe9c" strokeWidth="1.5" strokeLinecap="round" opacity="0.25" fill="none"/>
        <circle cx="148" cy="54" r="2.5" fill="#070d1f" opacity="0.4"/>
      </g>
      {/* Schooling fish trio */}
      <g style={{ opacity: 0.3 }}>
        <ellipse cx="100" cy="105" rx="8" ry="4" fill="#3adffa"/>
        <path d="M108 105 L114 102 L112 108 Z" fill="#3adffa"/>
        <ellipse cx="116" cy="100" rx="7" ry="3.5" fill="#3adffa"/>
        <path d="M123 100 L129 98 L127 103 Z" fill="#3adffa"/>
        <ellipse cx="126" cy="112" rx="7" ry="3.5" fill="#3adffa"/>
        <path d="M133 112 L139 110 L137 115 Z" fill="#3adffa"/>
      </g>
      {/* Sea star */}
      <g style={{ opacity: 0.28 }}>
        <path d="M28 125 L32 113 L36 125 L26 118 L38 118 Z" fill="#ff8796"/>
      </g>
      {/* Coral frag */}
      <g className="animate-sway" style={{ animationDelay: '2s' }}>
        <path d="M185 140 L185 118 M185 128 L178 118 M185 128 L192 116" stroke="#6dfe9c" strokeWidth="2" strokeLinecap="round" opacity="0.3"/>
      </g>
      {/* Bubbles */}
      {[
        { cx: 75, cy: 40, r: 3 }, { cx: 170, cy: 30, r: 2 }, { cx: 140, cy: 85, r: 2.5 },
        { cx: 45, cy: 95, r: 2 }, { cx: 200, cy: 80, r: 1.8 },
      ].map((b, i) => (
        <circle key={i} cx={b.cx} cy={b.cy} r={b.r} stroke="#3adffa" strokeWidth="1" opacity="0.2" fill="none"/>
      ))}
      {/* Sandy base */}
      <path d="M0 138 Q55 134 110 137 Q165 140 220 136 L220 140 L0 140 Z" fill="#ff8796" opacity="0.06"/>
    </svg>
  )
}

function ChemistrySilhouette() {
  return (
    <svg viewBox="0 0 220 140" fill="none" xmlns="http://www.w3.org/2000/svg" className="w-full h-full">
      {/* Flask 1 left */}
      <g>
        <path d="M42 60 L42 90 Q42 110 55 118 Q68 126 68 118 Q68 110 74 90 L74 60 Z" fill="#6dfe9c" opacity="0.1"/>
        <path d="M42 60 L74 60 M48 60 L48 52 L68 52 L68 60" stroke="#6dfe9c" strokeWidth="1.5" strokeLinecap="round" opacity="0.3" fill="none"/>
        <path d="M42 90 Q42 110 55 118 Q68 126 68 118 Q68 110 74 90" stroke="#6dfe9c" strokeWidth="1.5" opacity="0.3" fill="none"/>
        {/* Bubbles in flask */}
        {[{ cx: 52, cy: 105, r: 2.5 }, { cx: 62, cy: 95, r: 2 }, { cx: 56, cy: 82, r: 1.8 }].map((b, i) => (
          <circle key={i} cx={b.cx} cy={b.cy} r={b.r} stroke="#6dfe9c" strokeWidth="1" opacity="0.35" fill="none"/>
        ))}
      </g>
      {/* Flask 2 center */}
      <g>
        <path d="M88 55 L88 88 Q88 112 102 120 Q116 128 116 120 Q116 112 124 88 L124 55 Z" fill="#6dfe9c" opacity="0.08"/>
        <path d="M88 55 L124 55 M94 55 L94 46 L118 46 L118 55" stroke="#6dfe9c" strokeWidth="1.5" strokeLinecap="round" opacity="0.28" fill="none"/>
        <path d="M88 88 Q88 112 102 120 Q116 128 116 120 Q116 112 124 88" stroke="#6dfe9c" strokeWidth="1.5" opacity="0.28" fill="none"/>
        {[{ cx: 100, cy: 108, r: 3 }, { cx: 110, cy: 96, r: 2.2 }, { cx: 104, cy: 80, r: 2 }].map((b, i) => (
          <circle key={i} cx={b.cx} cy={b.cy} r={b.r} stroke="#6dfe9c" strokeWidth="1" opacity="0.3" fill="none"/>
        ))}
      </g>
      {/* Flask 3 right */}
      <g>
        <path d="M144 62 L144 91 Q144 110 156 118 Q168 126 168 118 Q168 110 174 91 L174 62 Z" fill="#6dfe9c" opacity="0.1"/>
        <path d="M144 62 L174 62 M150 62 L150 54 L168 54 L168 62" stroke="#6dfe9c" strokeWidth="1.5" strokeLinecap="round" opacity="0.3" fill="none"/>
        <path d="M144 91 Q144 110 156 118 Q168 126 168 118 Q168 110 174 91" stroke="#6dfe9c" strokeWidth="1.5" opacity="0.3" fill="none"/>
        {[{ cx: 153, cy: 106, r: 2.5 }, { cx: 162, cy: 94, r: 2 }].map((b, i) => (
          <circle key={i} cx={b.cx} cy={b.cy} r={b.r} stroke="#6dfe9c" strokeWidth="1" opacity="0.32" fill="none"/>
        ))}
      </g>
      {/* Rising bubble column far left */}
      {[{ cy: 125, r: 2 }, { cy: 108, r: 2.5 }, { cy: 90, r: 2 }, { cy: 74, r: 1.5 }].map((b, i) => (
        <circle key={i} cx="20" cy={b.cy} r={b.r} stroke="#6dfe9c" strokeWidth="1" opacity="0.2" fill="none"/>
      ))}
      {/* Rising bubble column far right */}
      {[{ cy: 128, r: 2 }, { cy: 112, r: 2.5 }, { cy: 95, r: 2 }].map((b, i) => (
        <circle key={i} cx="204" cy={b.cy} r={b.r} stroke="#6dfe9c" strokeWidth="1" opacity="0.2" fill="none"/>
      ))}
      {/* Molecule decoration */}
      <g style={{ opacity: 0.2 }}>
        <circle cx="195" cy="55" r="5" stroke="#6dfe9c" strokeWidth="1.5" fill="none"/>
        <circle cx="210" cy="45" r="4" stroke="#6dfe9c" strokeWidth="1.5" fill="none"/>
        <circle cx="205" cy="68" r="4" stroke="#6dfe9c" strokeWidth="1.5" fill="none"/>
        <line x1="200" y1="55" x2="206" y2="49" stroke="#6dfe9c" strokeWidth="1"/>
        <line x1="200" y1="58" x2="201" y2="64" stroke="#6dfe9c" strokeWidth="1"/>
      </g>
      <path d="M0 138 Q55 134 110 137 Q165 140 220 136 L220 140 L0 140 Z" fill="#6dfe9c" opacity="0.06"/>
    </svg>
  )
}

function AlertsSilhouette() {
  return (
    <svg viewBox="0 0 220 140" fill="none" xmlns="http://www.w3.org/2000/svg" className="w-full h-full">
      {/* Bleached/skeletal reef — sparse, colorless, stark */}
      {/* Bare branch left */}
      <g>
        <path d="M30 140 L30 100 M30 120 L20 108 M30 120 L40 106 M20 108 L14 96 M40 106 L46 94" stroke="#5a6080" strokeWidth="2" strokeLinecap="round" opacity="0.3"/>
      </g>
      {/* Dead staghorn center */}
      <g>
        <path d="M80 140 L80 105 M80 120 L68 106 M80 120 L92 104 M68 106 L62 92 M92 104 L98 90 M68 106 L60 96" stroke="#5a6080" strokeWidth="2.5" strokeLinecap="round" opacity="0.25"/>
      </g>
      {/* Fallen coral right of center */}
      <g>
        <path d="M130 136 L148 128 L160 120 M148 128 L144 118 M148 128 L154 116" stroke="#5a6080" strokeWidth="2" strokeLinecap="round" opacity="0.2"/>
      </g>
      {/* Bare stalk far right */}
      <g>
        <path d="M190 140 L190 108 M190 125 L182 114 M190 118 L198 110" stroke="#5a6080" strokeWidth="1.8" strokeLinecap="round" opacity="0.22"/>
      </g>
      {/* Scattered pebbles/rubble */}
      {[
        { cx: 52, cy: 136, rx: 8, ry: 3 }, { cx: 112, cy: 138, rx: 6, ry: 2.5 },
        { cx: 170, cy: 137, rx: 7, ry: 3 }, { cx: 210, cy: 138, rx: 5, ry: 2 },
      ].map((e, i) => (
        <ellipse key={i} cx={e.cx} cy={e.cy} rx={e.rx} ry={e.ry} fill="#5a6080" opacity="0.15"/>
      ))}
      {/* Sandy base */}
      <path d="M0 138 Q55 134 110 137 Q165 140 220 136 L220 140 L0 140 Z" fill="#5a6080" opacity="0.08"/>
    </svg>
  )
}

function HistorySilhouette() {
  return (
    <svg viewBox="0 0 220 140" fill="none" xmlns="http://www.w3.org/2000/svg" className="w-full h-full">
      {/* Kelp fronds — tall, varied heights */}
      {[
        { x: 20, h: 115, delay: '0s', anim: 'animate-sway' },
        { x: 40, h: 130, delay: '0.8s', anim: 'animate-sway2' },
        { x: 62, h: 108, delay: '1.6s', anim: 'animate-sway' },
        { x: 88, h: 125, delay: '0.4s', anim: 'animate-sway2' },
        { x: 112, h: 118, delay: '2s', anim: 'animate-sway' },
      ].map((k, i) => (
        <g key={i} className={k.anim} style={{ animationDelay: k.delay }}>
          <path
            d={`M${k.x} 140 C${k.x - 4} ${140 - k.h * 0.4} ${k.x + 6} ${140 - k.h * 0.7} ${k.x} ${140 - k.h}`}
            stroke="#3adffa"
            strokeWidth="2.5"
            strokeLinecap="round"
            opacity="0.28"
            fill="none"
          />
          {/* Frond blades */}
          <path
            d={`M${k.x} ${140 - k.h * 0.6} C${k.x - 10} ${140 - k.h * 0.65} ${k.x - 14} ${140 - k.h * 0.58} ${k.x - 8} ${140 - k.h * 0.55}`}
            stroke="#3adffa"
            strokeWidth="1.8"
            strokeLinecap="round"
            opacity="0.2"
            fill="none"
          />
          <path
            d={`M${k.x} ${140 - k.h * 0.75} C${k.x + 10} ${140 - k.h * 0.8} ${k.x + 14} ${140 - k.h * 0.73} ${k.x + 8} ${140 - k.h * 0.7}`}
            stroke="#3adffa"
            strokeWidth="1.8"
            strokeLinecap="round"
            opacity="0.2"
            fill="none"
          />
        </g>
      ))}
      {/* Jellyfish */}
      <g className="animate-jellyfish" style={{ animationDelay: '1s' }}>
        <path d="M170 40 Q175 28 180 25 Q185 22 190 25 Q195 28 200 40" fill="#3adffa" opacity="0.18"/>
        <path d="M172 40 C172 44 174 50 173 56" stroke="#3adffa" strokeWidth="1.2" strokeLinecap="round" opacity="0.15"/>
        <path d="M178 40 C178 46 180 54 179 62" stroke="#3adffa" strokeWidth="1.2" strokeLinecap="round" opacity="0.15"/>
        <path d="M184 40 C184 44 186 52 185 58" stroke="#3adffa" strokeWidth="1.2" strokeLinecap="round" opacity="0.15"/>
        <path d="M190 40 C190 46 188 54 190 60" stroke="#3adffa" strokeWidth="1.2" strokeLinecap="round" opacity="0.15"/>
        <path d="M196 40 C196 44 198 50 197 56" stroke="#3adffa" strokeWidth="1.2" strokeLinecap="round" opacity="0.15"/>
      </g>
      {/* Small kelp right side */}
      <g className="animate-sway" style={{ animationDelay: '3s' }}>
        <path d="M152 140 C149 120 153 105 150 88" stroke="#3adffa" strokeWidth="2" strokeLinecap="round" opacity="0.22" fill="none"/>
      </g>
      {/* Bubbles */}
      {[
        { cx: 30, cy: 15, r: 2.5 }, { cx: 90, cy: 8, r: 2 }, { cx: 145, cy: 72, r: 2 },
        { cx: 205, cy: 85, r: 1.8 }, { cx: 160, cy: 25, r: 1.5 },
      ].map((b, i) => (
        <circle key={i} cx={b.cx} cy={b.cy} r={b.r} stroke="#3adffa" strokeWidth="1" opacity="0.18" fill="none"/>
      ))}
      <path d="M0 138 Q55 134 110 137 Q165 140 220 136 L220 140 L0 140 Z" fill="#3adffa" opacity="0.06"/>
    </svg>
  )
}

const silhouetteMap: Record<PageKey, () => JSX.Element> = {
  dashboard: DashboardSilhouette,
  livestock: LivestockSilhouette,
  chemistry: ChemistrySilhouette,
  alerts: AlertsSilhouette,
  history: HistorySilhouette,
  default: DashboardSilhouette,
}

export function SidebarSilhouette({ page }: SidebarSilhouetteProps) {
  const Scene = silhouetteMap[page] ?? DashboardSilhouette
  return (
    <div className="absolute bottom-0 left-0 right-0 h-36 overflow-hidden pointer-events-none transition-fluid">
      <Scene />
    </div>
  )
}
