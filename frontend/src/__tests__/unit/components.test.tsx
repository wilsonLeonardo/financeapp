import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { Alert, StatCard, Modal, EmptyState } from '@/components/common'

describe('Alert', () => {
  it('renders error message', () => {
    render(<Alert message="Something went wrong" />)
    expect(screen.getByText('Something went wrong')).toBeInTheDocument()
  })

  it('calls onClose when X is clicked', () => {
    const onClose = vi.fn()
    render(<Alert message="Error" onClose={onClose} />)
    fireEvent.click(screen.getByRole('button'))
    expect(onClose).toHaveBeenCalledOnce()
  })

  it('does not render close button when onClose is not provided', () => {
    render(<Alert message="Error" />)
    expect(screen.queryByRole('button')).toBeNull()
  })
})

describe('StatCard', () => {
  it('renders label and value', () => {
    render(
      <StatCard label="Total Spent" value="$ 1.500,00" icon={<span>💸</span>} />
    )
    expect(screen.getByText('Total Spent')).toBeInTheDocument()
    expect(screen.getByText('$ 1.500,00')).toBeInTheDocument()
  })

  it('renders sub text when provided', () => {
    render(
      <StatCard label="Label" value="Value" sub="+10% vs mês anterior" icon={<span />} trend="up" />
    )
    expect(screen.getByText('+10% vs mês anterior')).toBeInTheDocument()
  })
})

describe('Modal', () => {
  it('renders children when open', () => {
    render(
      <Modal open={true} onClose={() => {}} title="Test Modal">
        <p>Modal Content</p>
      </Modal>
    )
    expect(screen.getByText('Test Modal')).toBeInTheDocument()
    expect(screen.getByText('Modal Content')).toBeInTheDocument()
  })

  it('does not render when closed', () => {
    render(
      <Modal open={false} onClose={() => {}} title="Hidden Modal">
        <p>Hidden Content</p>
      </Modal>
    )
    expect(screen.queryByText('Hidden Modal')).toBeNull()
  })

  it('calls onClose when backdrop is clicked', () => {
    const onClose = vi.fn()
    const { container } = render(
      <Modal open={true} onClose={onClose} title="Modal">
        <p>Content</p>
      </Modal>
    )
    // Click the backdrop (first absolute div)
    const backdrop = container.querySelector('.absolute.inset-0')
    if (backdrop) fireEvent.click(backdrop)
    expect(onClose).toHaveBeenCalled()
  })
})

describe('EmptyState', () => {
  it('renders message', () => {
    render(<EmptyState message="No data found" />)
    expect(screen.getByText('No data found')).toBeInTheDocument()
  })

  it('renders icon when provided', () => {
    render(<EmptyState message="Empty" icon="📭" />)
    expect(screen.getByText('📭')).toBeInTheDocument()
  })
})
