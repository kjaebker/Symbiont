import { useState } from 'react'
import { usePageTitle } from '@/hooks/usePageTitle'
import { PageHeader } from '@/components/PageHeader'
import { tabs, type Tab } from './settings/_shared'
import DashboardTab from './settings/DashboardTab'
import DevicesTab from './settings/DevicesTab'
import TankTab from './settings/TankTab'
import ProbesTab from './settings/ProbesTab'
import OutletsTab from './settings/OutletsTab'
import AgentTab from './settings/AgentTab'
import TokensTab from './settings/TokensTab'
import NotificationsTab from './settings/NotificationsTab'
import BackupTab from './settings/BackupTab'
import SystemLogTab from './settings/SystemLogTab'
import SystemTab from './settings/SystemTab'

const mobileCardTabs: Tab[] = ['probes', 'outlets', 'tokens', 'notifications']
const noContainerTabs: Tab[] = ['tank', 'system']

export default function Settings() {
  usePageTitle('Settings')
  const [activeTab, setActiveTab] = useState<Tab>('dashboard')

  const tabContainerClass = noContainerTabs.includes(activeTab)
    ? undefined
    : mobileCardTabs.includes(activeTab)
    ? 'sm:bg-surface-container sm:rounded-2xl sm:overflow-hidden'
    : 'bg-surface-container rounded-2xl overflow-hidden'

  return (
    <div>
      <PageHeader
        subtitle="Admin Console"
        title="System Core Settings"
        tabs={tabs.map((t) => ({ key: t.key, label: t.label }))}
        activeTab={activeTab}
        onTabChange={(k) => setActiveTab(k as Tab)}
        maxWidth="max-w-7xl"
      />

      <div className="px-6 md:px-8 pt-5 pb-8 max-w-7xl mx-auto">
        <div className={tabContainerClass}>
          {activeTab === 'dashboard' && <DashboardTab />}
          {activeTab === 'devices' && <DevicesTab />}
          {activeTab === 'tank' && <TankTab />}
          {activeTab === 'probes' && <ProbesTab />}
          {activeTab === 'outlets' && <OutletsTab />}
          {activeTab === 'agent' && <AgentTab />}
          {activeTab === 'tokens' && <TokensTab />}
          {activeTab === 'notifications' && <NotificationsTab />}
          {activeTab === 'backup' && <BackupTab />}
          {activeTab === 'log' && <SystemLogTab />}
          {activeTab === 'system' && <SystemTab />}
        </div>
      </div>
    </div>
  )
}
