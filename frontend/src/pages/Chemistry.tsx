import { useState } from 'react'
import { Navigate, useSearchParams } from 'react-router-dom'
import { Plus } from 'lucide-react'
import { usePageTitle } from '@/hooks/usePageTitle'
import Measurements from '@/pages/Measurements'
import { PageHeader } from '@/components/PageHeader'
import { FilterDropdown } from '@/components/FilterDropdown'
import { useMeasurementParameters } from '@/hooks/useMeasurements'

// ─── Chemistry Page ───────────────────────────────────────────────────────────

export default function Chemistry() {
  const [searchParams] = useSearchParams()
  usePageTitle('Chemistry')

  const { data: paramsData } = useMeasurementParameters()
  const parameters = paramsData?.parameters ?? []

  const [selectedParam, setSelectedParam] = useState<string | null>(null)
  const [showMeasurementForm, setShowMeasurementForm] = useState(false)

  // Dosing used to be a tab here. Logging doses is on Tasks now; schedules
  // and the product catalog are in Settings.
  if (searchParams.get('tab') === 'dosing') return <Navigate to="/tasks" replace />

  const measurementFilterBar = (
    <FilterDropdown
      label="All parameters"
      value={selectedParam ?? ''}
      options={[
        { value: '', label: 'All parameters' },
        ...parameters.map((p) => ({ value: p.name, label: p.name })),
      ]}
      onChange={(v) => setSelectedParam(v === '' ? null : v)}
    />
  )

  return (
    <div>
      <PageHeader
        subtitle="Chemical Balance"
        title="Water Chemistry"
        action={{ label: 'Add Measurement', icon: Plus, onClick: () => setShowMeasurementForm((v) => !v), active: showMeasurementForm }}
        filterBar={measurementFilterBar}
        filterCount={1}
        filterActive={!!selectedParam}
        onClearFilters={() => setSelectedParam(null)}
      />

      <Measurements
        hideHeader
        selectedParam={selectedParam}
        onSelectedParamChange={setSelectedParam}
        showForm={showMeasurementForm}
        onShowFormChange={setShowMeasurementForm}
      />
    </div>
  )
}
