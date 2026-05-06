import { useState } from 'react'
import { cn } from '@/lib/utils'
import { Plus,Copy,Check,Sparkles,Bot,ChevronDown,ChevronUp } from 'lucide-react'
import { useCreateToken } from '@/hooks/useSettings'
import { useAgentSettings,useUpdateAgentSettings,useAgentContext,useAgentSkills } from '@/hooks/useAgent'
import { LoadingState,EmptyState } from './_shared'

// =============================================================================
// Agent Tab
// =============================================================================

export default function AgentTab() {
  const { data: settings, isLoading: settingsLoading } = useAgentSettings()
  const { data: skillsData, isLoading: skillsLoading } = useAgentSkills()
  const { data: contextData, isLoading: contextLoading, refetch: refetchContext } = useAgentContext()
  const updateSettings = useUpdateAgentSettings()
  const createToken = useCreateToken()
  const [showContext, setShowContext] = useState(false)
  const [copied, setCopied] = useState(false)
  const [personaCopied, setPersonaCopied] = useState(false)
  const [connectorToken, setConnectorToken] = useState<string | null>(null)
  const [connectorCopied, setConnectorCopied] = useState(false)

  if (settingsLoading || skillsLoading) return <LoadingState label="Loading agent settings..." />
  if (!settings) return null

  const skills = skillsData?.skills ?? []
  const context = contextData?.context ?? ''

  function handleToneChange(tone: string) {
    updateSettings.mutate({ tone: tone as 'analytical' | 'casual' | 'terse' })
  }

  function handleProductLineChange(line: string) {
    updateSettings.mutate({ dosing_product_line: line as 'brs_pharma' | 'red_sea' | 'tropic_marin' | 'generic' | 'none' })
  }

  function handleVolumeChange(val: string) {
    const n = parseFloat(val)
    if (val === '') {
      updateSettings.mutate({ net_volume_gallons: null })
    } else if (!isNaN(n) && n > 0) {
      updateSettings.mutate({ net_volume_gallons: n })
    }
  }

  function handleGuardrailsChange(val: string) {
    updateSettings.mutate({ custom_guardrails: val || null })
  }

  function handleSkillToggle(name: string, enabled: boolean) {
    const allNames = skills.map((s) => s.name)
    // Expand empty list (opt-out "all enabled") to explicit list before mutating,
    // otherwise filtering an empty array always stays empty.
    const current =
      settings!.enabled_skills && settings!.enabled_skills.length > 0
        ? settings!.enabled_skills
        : allNames
    let next: string[]
    if (enabled) {
      next = current.includes(name) ? current : [...current, name]
    } else {
      next = current.filter((n) => n !== name)
    }
    // Collapse back to empty when all skills are on (opt-out model)
    if (next.length === allNames.length) next = []
    updateSettings.mutate({ enabled_skills: next })
  }

  function isSkillEnabled(name: string): boolean {
    if (!settings!.enabled_skills || settings!.enabled_skills.length === 0) return true
    return settings!.enabled_skills.includes(name)
  }

  async function handleCopyContext() {
    await refetchContext()
    if (context) {
      navigator.clipboard.writeText(context).then(() => {
        setCopied(true)
        setTimeout(() => setCopied(false), 2000)
      })
    }
  }

  async function handleCopyPersona() {
    const { data } = await refetchContext()
    const text = data?.context ?? context
    if (text) {
      navigator.clipboard.writeText(text).then(() => {
        setPersonaCopied(true)
        setTimeout(() => setPersonaCopied(false), 2000)
      })
    }
  }

  function handleCreateConnectorToken() {
    createToken.mutate({ label: 'claude-mobile', scope: 'control' }, {
      onSuccess: (data) => setConnectorToken(data.token),
    })
  }

  function handleCopyConnectorToken(token: string) {
    navigator.clipboard.writeText(token).then(() => {
      setConnectorCopied(true)
      setTimeout(() => setConnectorCopied(false), 2000)
    })
  }

  const selectClass = 'w-full bg-surface-container-highest text-on-surface text-base rounded-xl px-3 py-2.5 outline-none focus:ring-2 focus:ring-primary/40 transition-fluid'
  const labelClass = 'text-xs text-on-surface-faint uppercase tracking-widest font-medium mb-1.5 block'

  return (
    <div className="space-y-6 p-4">

      {/* Header */}
      <div className="flex items-center gap-3 px-1 pt-1">
        <div className="h-8 w-8 rounded-xl bg-primary/10 flex items-center justify-center shrink-0">
          <Bot size={16} className="text-primary" />
        </div>
        <div>
          <p className="text-sm font-semibold text-on-surface">AI Assistant Persona</p>
          <p className="text-xs text-on-surface-faint">Configure how Claude responds to your tank data</p>
        </div>
      </div>

      {/* Persona controls */}
      <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">

        {/* Tone */}
        <div>
          <label className={labelClass}>Tone</label>
          <select
            value={settings.tone}
            onChange={(e) => handleToneChange(e.target.value)}
            className={selectClass}
          >
            <option value="analytical">Analytical — precise, data-driven</option>
            <option value="casual">Casual — conversational, friendly</option>
            <option value="terse">Terse — minimal, just the facts</option>
          </select>
        </div>

        {/* Dosing product line */}
        <div>
          <label className={labelClass}>Dosing Product Line</label>
          <select
            value={settings.dosing_product_line}
            onChange={(e) => handleProductLineChange(e.target.value)}
            className={selectClass}
          >
            <option value="generic">Generic (brand-neutral)</option>
            <option value="brs_pharma">BRS Pharma</option>
            <option value="red_sea">Red Sea</option>
            <option value="tropic_marin">Tropic Marin</option>
            <option value="none">None (no dosing advice)</option>
          </select>
        </div>

        {/* Net volume override */}
        <div>
          <label className={labelClass}>Net Volume Override (gallons)</label>
          <input
            type="number"
            min={1}
            step={0.5}
            placeholder="Derived from tank profile"
            defaultValue={settings.net_volume_gallons ?? ''}
            onBlur={(e) => handleVolumeChange(e.target.value)}
            className={cn(selectClass, 'placeholder:text-on-surface-faint/50')}
          />
          <p className="text-xs text-on-surface-faint mt-1">
            Leave blank to use the sum of display + sump volumes
          </p>
        </div>
      </div>

      {/* Custom guardrails */}
      <div>
        <label className={labelClass}>Custom Guardrails</label>
        <textarea
          rows={3}
          placeholder="e.g. Never recommend Alk above 10 dKH for this tank."
          defaultValue={settings.custom_guardrails ?? ''}
          onBlur={(e) => handleGuardrailsChange(e.target.value)}
          className={cn(selectClass, 'resize-y placeholder:text-on-surface-faint/50')}
        />
        <p className="text-xs text-on-surface-faint mt-1">
          Appended verbatim to every context block sent to Claude
        </p>
      </div>

      {/* Skills */}
      <div>
        <p className={labelClass}>Skills</p>
        <p className="text-xs text-on-surface-faint mb-3">
          Toggle skills to include or exclude them from installs. All skills are enabled by default.
        </p>
        {skillsLoading ? (
          <LoadingState label="Loading skills..." />
        ) : skills.length === 0 ? (
          <EmptyState icon={<Sparkles size={24} />} message="No skills available." />
        ) : (
          <div className="space-y-1">
            {skills.map((skill) => {
              const on = isSkillEnabled(skill.name)
              return (
                <div
                  key={skill.name}
                  className="flex items-start gap-3 px-3 py-3 rounded-xl bg-surface-container-high/40 hover:bg-surface-container-high/70 transition-fluid"
                >
                  <button
                    onClick={() => handleSkillToggle(skill.name, !on)}
                    className={cn(
                      'mt-0.5 h-4 w-4 rounded-full shrink-0 border-2 transition-fluid',
                      on
                        ? 'bg-primary border-primary'
                        : 'bg-transparent border-on-surface-faint/40',
                    )}
                    aria-label={on ? `Disable ${skill.name}` : `Enable ${skill.name}`}
                  />
                  <div className="min-w-0 flex-1">
                    <p className="text-sm font-medium text-on-surface leading-snug">{skill.name}</p>
                    <p className="text-xs text-on-surface-faint leading-snug mt-0.5">{skill.description}</p>
                  </div>
                </div>
              )
            })}
          </div>
        )}
      </div>

      {/* Install instructions */}
      <div className="rounded-2xl bg-surface-container-high/40 p-4 space-y-2">
        <p className="text-xs text-on-surface-faint uppercase tracking-widest font-medium">Install Skills</p>
        <p className="text-sm text-on-surface-dim">
          Run this command to install enabled skills into your Claude Code skills directory:
        </p>
        <div className="flex items-center gap-2 bg-surface-container-highest rounded-xl px-3 py-2 font-mono text-xs text-primary">
          <span className="flex-1 select-all">symbiont skills install</span>
          <span className="text-on-surface-faint">→ ~/.claude/skills/symbiont/</span>
        </div>
        <p className="text-xs text-on-surface-faint">
          Claude Code auto-discovers skills in that directory. Re-run after toggling skills or updating Symbiont.
        </p>
      </div>

      {/* claude.ai Connector */}
      <div className="rounded-2xl bg-surface-container-high/40 p-4 space-y-4">
        <div className="flex items-center gap-2">
          <p className="text-xs text-on-surface-faint uppercase tracking-widest font-medium">claude.ai Connector</p>
        </div>
        <p className="text-sm text-on-surface-dim">
          Add Symbiont as a remote MCP connector in a claude.ai Project for mobile access.
          The MCP endpoint is at <code className="text-primary font-mono text-xs">/api/mcp</code> on your Symbiont host.
        </p>

        {/* Step 1: Persona */}
        <div className="space-y-1.5">
          <p className="text-xs text-on-surface-dim font-medium">1. Copy your agent persona</p>
          <p className="text-xs text-on-surface-faint">Paste into the Project&apos;s Custom Instructions so Claude knows your tank without calling tools every turn.</p>
          <button
            onClick={handleCopyPersona}
            className="flex items-center gap-2 px-3 py-2 rounded-xl text-xs font-semibold text-on-surface-dim bg-surface-container-highest hover:bg-surface-container-high transition-fluid cursor-pointer"
          >
            {personaCopied ? <Check size={13} className="text-secondary" /> : <Copy size={13} />}
            {personaCopied ? 'Persona copied!' : 'Copy persona for claude.ai'}
          </button>
        </div>

        {/* Step 2: Connector token */}
        <div className="space-y-1.5">
          <p className="text-xs text-on-surface-dim font-medium">2. Create a connector token</p>
          <p className="text-xs text-on-surface-faint">Creates a <code className="text-primary font-mono">control</code>-scoped token labeled <code className="text-primary font-mono">claude-mobile</code>. Use it as the Bearer token in the claude.ai connector settings.</p>
          {connectorToken ? (
            <div className="space-y-2">
              <div className="flex items-center gap-2 bg-surface-container-highest rounded-xl px-3 py-2">
                <code className="flex-1 text-xs text-secondary font-mono break-all">{connectorToken}</code>
                <button
                  onClick={() => handleCopyConnectorToken(connectorToken)}
                  className="p-1 rounded-lg text-on-surface-faint hover:text-secondary hover:bg-secondary/10 transition-fluid cursor-pointer shrink-0"
                >
                  {connectorCopied ? <Check size={13} /> : <Copy size={13} />}
                </button>
              </div>
              <p className="text-xs text-tertiary">Save this token — it won&apos;t be shown again.</p>
              <button
                onClick={() => setConnectorToken(null)}
                className="text-xs text-on-surface-faint hover:text-on-surface transition-fluid cursor-pointer"
              >
                Dismiss
              </button>
            </div>
          ) : (
            <button
              onClick={handleCreateConnectorToken}
              disabled={createToken.isPending}
              className="flex items-center gap-2 px-3 py-2 rounded-xl text-xs font-semibold text-on-primary bg-gradient-to-r from-primary to-primary-container hover:shadow-glow-primary transition-fluid cursor-pointer disabled:opacity-50"
            >
              <Plus size={13} />
              Create connector token
            </button>
          )}
        </div>

        {/* Step 3: Walkthrough */}
        <div className="space-y-1.5">
          <p className="text-xs text-on-surface-dim font-medium">3. Add connector in claude.ai</p>
          <ol className="text-xs text-on-surface-faint space-y-1 list-decimal list-inside">
            <li>Open a Project in claude.ai → <span className="text-on-surface-dim">Project settings → Connectors</span></li>
            <li>Click <span className="text-on-surface-dim">Add connector → Custom MCP server</span></li>
            <li>Enter your Symbiont URL and the token above</li>
            <li>Paste the persona into <span className="text-on-surface-dim">Custom Instructions</span></li>
          </ol>
          <p className="text-xs text-on-surface-faint pt-1">
            Need a public URL?{' '}
            <span className="text-on-surface-dim">Use Tailscale Funnel or Cloudflare Tunnel — see <code className="text-primary font-mono">docs/deployment-remote-mcp.md</code>.</span>
          </p>
        </div>
      </div>

      {/* Context preview */}
      <div>
        <button
          onClick={() => { setShowContext((v) => !v); if (!showContext) refetchContext() }}
          className="flex items-center gap-2 text-xs text-on-surface-dim hover:text-on-surface transition-fluid"
        >
          {showContext ? <ChevronUp size={14} /> : <ChevronDown size={14} />}
          <span className="uppercase tracking-widest font-medium">Preview assembled context</span>
        </button>

        {showContext && (
          <div className="mt-3 space-y-2">
            <div className="flex justify-end">
              <button
                onClick={handleCopyContext}
                className="flex items-center gap-1.5 px-3 py-1.5 rounded-xl text-xs font-semibold text-on-surface-dim bg-surface-container-high hover:bg-surface-container-highest transition-fluid"
              >
                {copied ? <Check size={13} className="text-secondary" /> : <Copy size={13} />}
                {copied ? 'Copied!' : 'Copy'}
              </button>
            </div>
            {contextLoading ? (
              <LoadingState label="Building context..." />
            ) : (
              <pre className="bg-surface-container-highest rounded-xl p-4 text-xs text-on-surface-dim overflow-auto max-h-96 whitespace-pre-wrap leading-relaxed font-mono">
                {context || '(empty)'}
              </pre>
            )}
          </div>
        )}
      </div>
    </div>
  )
}

