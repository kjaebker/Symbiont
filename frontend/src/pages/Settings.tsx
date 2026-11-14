import { useSearchParams } from 'react-router-dom'
import { usePageTitle } from '@/hooks/usePageTitle'
import { PageHeader } from '@/components/PageHeader'
import { PageBody } from '@/components/ui'
import { cn } from '@/lib/utils'
import { tabGroups, type Tab, type TabGroup } from './settings/_shared'
import DashboardTab from './settings/DashboardTab'
import DevicesTab from './settings/DevicesTab'
import TankTab from './settings/TankTab'
import ProbesTab from './settings/ProbesTab'
import OutletsTab from './settings/OutletsTab'
import DosingTab from './settings/DosingTab'
import AlertsTab from './settings/AlertsTab'
import AgentTab from './settings/AgentTab'
import TokensTab from './settings/TokensTab'
import NotificationsTab from './settings/NotificationsTab'
import BackupTab from './settings/BackupTab'
import SystemLogTab from './settings/SystemLogTab'
import SystemTab from './settings/SystemTab'

const mobileCardTabs: Tab[] = ['probes', 'outlets', 'tokens', 'notifications']
const noContainerTabs: Tab[] = ['tank', 'system', 'dosing', 'alerts']

export default function Settings() {
  usePageTitle('Settings')
  const [searchParams, setSearchParams] = useSearchParams()
  const tabParam = searchParams.get('tab')
  const activeGroup =
    tabGroups.find((g) => g.tabs.some((t) => t.key === tabParam)) ?? tabGroups[0]
  const activeTab: Tab = activeGroup.tabs.find((t) => t.key === tabParam)?.key ?? activeGroup.tabs[0].key
  const setActiveTab = (t: Tab) => setSearchParams({ tab: t }, { replace: true })
  const setActiveGroup = (g: TabGroup) => {
    const group = tabGroups.find((x) => x.key === g)
    if (group) setActiveTab(group.tabs[0].key)
  }

  const tabContainerClass = noContainerTabs.includes(activeTab)
    ? undefined
    : mobileCardTabs.includes(activeTab)
    ? 'sm:bg-surface-container sm:rounded-2xl sm:overflow-hidden'
    : 'bg-surface-container rounded-2xl overflow-hidden'

  return (
    <div>
      <PageHeader
        subtitle="Configuration"
        title="Settings"
        tabs={tabGroups.map((g) => ({ key: g.key, label: g.label }))}
        activeTab={activeGroup.key}
        onTabChange={(k) => setActiveGroup(k as TabGroup)}
      />

      <PageBody>
        {/* Second level: the tabs inside the active group */}
        {activeGroup.tabs.length > 1 && (
          <div className="flex flex-wrap gap-1.5" role="tablist" aria-label={`${activeGroup.label} settings`}>
            {activeGroup.tabs.map((t) => {
              const isActive = t.key === activeTab
              return (
                <button
                  key={t.key}
                  role="tab"
                  aria-selected={isActive}
                  onClick={() => setActiveTab(t.key)}
                  className={cn(
                    'px-3.5 py-1.5 rounded-full text-xs font-semibold uppercase tracking-wider cursor-pointer transition-fluid',
                    isActive
                      ? 'bg-surface-container-highest text-on-surface'
                      : 'bg-surface-container text-on-surface-dim hover:text-on-surface hover:bg-surface-container-high',
                  )}
                >
                  {t.label}
                </button>
              )
            })}
          </div>
        )}

        <div className={tabContainerClass}>
          {activeTab === 'dashboard' && <DashboardTab />}
          {activeTab === 'tank' && <TankTab />}
          {activeTab === 'probes' && <ProbesTab />}
          {activeTab === 'outlets' && <OutletsTab />}
          {activeTab === 'devices' && <DevicesTab />}
          {activeTab === 'dosing' && <DosingTab />}
          {activeTab === 'alerts' && <AlertsTab />}
          {activeTab === 'notifications' && <NotificationsTab />}
          {activeTab === 'system' && <SystemTab />}
          {activeTab === 'backup' && <BackupTab />}
          {activeTab === 'tokens' && <TokensTab />}
          {activeTab === 'agent' && <AgentTab />}
          {activeTab === 'log' && <SystemLogTab />}
        </div>
      </PageBody>
    </div>
  )
}
