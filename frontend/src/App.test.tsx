import { cleanup, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import App from './App'

describe('calculator', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn())
  })

  afterEach(() => {
    cleanup()
    vi.restoreAllMocks()
  })

  it('renders the calculator form with binary inputs', () => {
    render(<App />)

    expect(screen.getByRole('heading', { name: /make a calculation/i })).toBeInTheDocument()
    expect(screen.getByLabelText(/first number/i)).toBeInTheDocument()
    expect(screen.getByLabelText(/second number/i)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /calculate/i })).toBeDisabled()
    expect(screen.getByLabelText(/first number/i)).toHaveAttribute('aria-invalid', 'true')
    expect(screen.getByLabelText(/first number/i)).toHaveAttribute('aria-describedby', 'validation-error')
  })

  it('submits operands and displays a result', async () => {
    const fetchMock = vi.mocked(fetch)
    fetchMock.mockResolvedValue(
      new Response(JSON.stringify({ result: 15 }), { status: 200 }),
    )
    const user = userEvent.setup()
    render(<App />)

    await user.type(screen.getByLabelText(/first number/i), '12')
    await user.type(screen.getByLabelText(/second number/i), '3')
    await user.click(screen.getByRole('button', { name: /calculate/i }))

    expect(fetchMock).toHaveBeenCalledWith(
      '/api/calculate',
      expect.objectContaining({
        method: 'POST',
        body: JSON.stringify({ operation: 'add', operands: [12, 3] }),
      }),
    )
    expect(await screen.findByText('15')).toBeInTheDocument()
  })

  it('disables operation and operands while a calculation is in flight', async () => {
    let releaseResponse!: () => void
    const responseReady = new Promise<void>((resolve) => {
      releaseResponse = resolve
    })
    const fetchMock = vi.mocked(fetch)
    fetchMock.mockImplementation(async () => {
      await responseReady
      return new Response(JSON.stringify({ result: 15 }), { status: 200 })
    })
    const user = userEvent.setup()
    render(<App />)

    await user.type(screen.getByLabelText(/first number/i), '12')
    await user.type(screen.getByLabelText(/second number/i), '3')
    await user.click(screen.getByRole('button', { name: /calculate/i }))

    expect(screen.getByLabelText('Operation')).toBeDisabled()
    expect(screen.getByLabelText(/first number/i)).toBeDisabled()
    expect(screen.getByLabelText(/second number/i)).toBeDisabled()

    releaseResponse()
    expect(await screen.findByText('15')).toBeInTheDocument()
    expect(screen.getByLabelText(/first number/i)).toBeEnabled()
  })

  it('formats very large results with readable scientific notation', async () => {
    const fetchMock = vi.mocked(fetch)
    fetchMock.mockResolvedValue(
      new Response(JSON.stringify({ result: 1.0715086071862673e301 }), { status: 200 }),
    )
    const user = userEvent.setup()
    render(<App />)

    await user.type(screen.getByLabelText(/first number/i), '2')
    await user.type(screen.getByLabelText(/second number/i), '1000')
    await user.click(screen.getByRole('button', { name: /calculate/i }))

    expect(await screen.findByText('1.071509e+301')).toBeInTheDocument()
  })

  it('shows only one input and blocks invalid values', async () => {
    const user = userEvent.setup()
    render(<App />)

    await user.selectOptions(screen.getByLabelText('Operation'), 'sqrt')
    expect(screen.queryByLabelText(/second number/i)).not.toBeInTheDocument()
    await user.type(screen.getByLabelText('Number'), '-4')

    expect(screen.getByText(/square root requires a non-negative number/i)).toBeInTheDocument()
    expect(screen.getByLabelText('Number')).toHaveAttribute('aria-invalid', 'true')
    expect(screen.getByLabelText('Number')).toHaveAttribute('aria-describedby', 'validation-error')
    expect(screen.getByRole('button', { name: /calculate/i })).toBeDisabled()
  })

  it('shows a validation error for division by zero', async () => {
    const user = userEvent.setup()
    render(<App />)

    await user.selectOptions(screen.getByLabelText('Operation'), 'divide')
    await user.type(screen.getByLabelText(/first number/i), '8')
    await user.type(screen.getByLabelText(/second number/i), '0')

    expect(screen.getByText(/divisor cannot be zero/i)).toBeInTheDocument()
    expect(screen.getByLabelText(/second number/i)).toHaveAttribute('aria-invalid', 'true')
    expect(screen.getByLabelText(/second number/i)).toHaveAttribute('aria-describedby', 'validation-error')
    expect(screen.getByLabelText(/first number/i)).toHaveAttribute('aria-invalid', 'false')
    expect(screen.getByRole('button', { name: /calculate/i })).toBeDisabled()
  })

  it('surfaces backend error messages', async () => {
    const fetchMock = vi.mocked(fetch)
    fetchMock.mockResolvedValue(
      new Response(
        JSON.stringify({
          error: { code: 'DIVISION_BY_ZERO', message: 'Cannot divide by zero.' },
        }),
        { status: 422 },
      ),
    )
    const user = userEvent.setup()
    render(<App />)

    await user.selectOptions(screen.getByLabelText('Operation'), 'divide')
    await user.type(screen.getByLabelText(/first number/i), '8')
    await user.type(screen.getByLabelText(/second number/i), '2')
    await user.click(screen.getByRole('button', { name: /calculate/i }))

    expect(await screen.findByRole('alert')).toHaveTextContent('Cannot divide by zero.')
  })

  it('handles an empty backend response without a JSON parsing exception', async () => {
    const fetchMock = vi.mocked(fetch)
    fetchMock.mockResolvedValue(new Response('', { status: 502 }))
    const user = userEvent.setup()
    render(<App />)

    await user.type(screen.getByLabelText(/first number/i), '8')
    await user.type(screen.getByLabelText(/second number/i), '2')
    await user.click(screen.getByRole('button', { name: /calculate/i }))

    expect(await screen.findByRole('alert')).toHaveTextContent(
      'The calculation could not be completed.',
    )
  })

  it('reports invalid non-empty backend responses as invalid JSON', async () => {
    const fetchMock = vi.mocked(fetch)
    fetchMock.mockResolvedValue(new Response('not-json', { status: 502 }))
    const user = userEvent.setup()
    render(<App />)

    await user.type(screen.getByLabelText(/first number/i), '8')
    await user.type(screen.getByLabelText(/second number/i), '2')
    await user.click(screen.getByRole('button', { name: /calculate/i }))

    expect(await screen.findByRole('alert')).toHaveTextContent(
      'The calculator service returned invalid JSON.',
    )
  })
})
