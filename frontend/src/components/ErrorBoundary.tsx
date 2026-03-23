import { Component, type ReactNode } from 'react'
import { AlertTriangle, RefreshCw } from 'lucide-react'

interface Props {
  children: ReactNode
}

interface State {
  hasError: boolean
  error: Error | null
}

export class ErrorBoundary extends Component<Props, State> {
  state: State = { hasError: false, error: null }

  static getDerivedStateFromError(error: Error): State {
    return { hasError: true, error }
  }

  handleRetry = () => {
    this.setState({ hasError: false, error: null })
  }

  render() {
    if (this.state.hasError) {
      return (
        <div className="flex items-center justify-center min-h-[400px] p-8">
          <div className="bg-surface-container rounded-2xl p-8 max-w-md w-full text-center space-y-4">
            <div className="w-12 h-12 rounded-xl bg-tertiary/10 flex items-center justify-center mx-auto">
              <AlertTriangle size={24} className="text-tertiary" />
            </div>
            <h2 className="text-lg font-semibold text-on-surface">
              Something went wrong
            </h2>
            <p className="text-sm text-on-surface-dim">
              {this.state.error?.message || 'An unexpected error occurred.'}
            </p>
            <button
              onClick={this.handleRetry}
              className="inline-flex items-center gap-2 px-4 py-2 rounded-xl bg-surface-container-high text-on-surface text-sm font-medium hover:bg-surface-container-highest transition-fluid"
            >
              <RefreshCw size={14} />
              Try again
            </button>
          </div>
        </div>
      )
    }

    return this.props.children
  }
}
