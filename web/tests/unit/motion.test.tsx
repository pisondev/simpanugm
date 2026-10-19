import { act, render, screen } from "@testing-library/react"
import { afterEach, describe, expect, it, vi } from "vitest"

import { AnimatedBar, CountUp, PageTransition, Reveal, useReducedMotion } from "@/components/motion/primitives"

function preference(initial: boolean) {
  let reduced = initial
  const listeners = new Set<() => void>()
  vi.stubGlobal("matchMedia", () => ({
    get matches() {
      return reduced
    },
    addEventListener: (_: string, listener: () => void) => listeners.add(listener),
    removeEventListener: (_: string, listener: () => void) => listeners.delete(listener),
  }))
  return {
    change: (value: boolean) =>
      act(() => {
        reduced = value
        listeners.forEach((listener) => listener())
      }),
    listeners,
  }
}

afterEach(() => vi.unstubAllGlobals())

describe("motion accessibility", () => {
  it("renders final counts and bar values without scheduling motion when reduced", () => {
    preference(true)
    const frame = vi.fn()
    const animate = vi.fn()
    vi.stubGlobal("requestAnimationFrame", frame)
    vi.stubGlobal("IntersectionObserver", undefined)
    const original = HTMLElement.prototype.animate
    HTMLElement.prototype.animate = animate
    const { container, unmount } = render(
      <>
        <CountUp value={81} suffix="%" />
        <AnimatedBar value={65} />
      </>,
    )
    expect(container.querySelector(".sr-only")).toHaveTextContent("81%")
    expect(container.querySelector("[style]")).toHaveStyle({ transform: "scaleX(0.65)" })
    expect(frame).not.toHaveBeenCalled()
    expect(animate).not.toHaveBeenCalled()
    unmount()
    HTMLElement.prototype.animate = original
  })

  it("cancels an active count and restores the final value when preference changes", () => {
    const media = preference(false)
    vi.stubGlobal("IntersectionObserver", undefined)
    vi.stubGlobal(
      "requestAnimationFrame",
      vi.fn(() => 7),
    )
    const cancel = vi.fn()
    vi.stubGlobal("cancelAnimationFrame", cancel)
    const { container, unmount } = render(<CountUp value={92} suffix="%" />)
    media.change(true)
    expect(cancel).toHaveBeenCalledWith(7)
    expect(container.querySelector("span[aria-hidden]:last-child")).toHaveTextContent("92%")
    unmount()
    expect(media.listeners.size).toBe(0)
  })

  it("animates only after intersection and disconnects its observer", () => {
    preference(false)
    const disconnect = vi.fn()
    const observe = vi.fn()
    let intersect: (entries: { isIntersecting: boolean }[]) => void = () => undefined
    vi.stubGlobal(
      "IntersectionObserver",
      class {
        constructor(callback: typeof intersect) {
          intersect = callback
        }
        observe = observe
        disconnect = disconnect
      },
    )
    const frame = vi.fn(() => 1)
    vi.stubGlobal("requestAnimationFrame", frame)
    vi.stubGlobal("cancelAnimationFrame", vi.fn())
    const { unmount } = render(<CountUp value={70} />)
    expect(observe).toHaveBeenCalledOnce()
    expect(frame).not.toHaveBeenCalled()
    act(() => intersect([{ isIntersecting: true }]))
    expect(frame).toHaveBeenCalledOnce()
    expect(disconnect).toHaveBeenCalled()
    unmount()
  })

  it("keeps content present and works without browser motion APIs", () => {
    vi.stubGlobal("matchMedia", undefined)
    function Status() {
      return <span>{String(useReducedMotion())}</span>
    }
    render(
      <PageTransition>
        <Reveal>
          <button>Continue</button>
          <Status />
        </Reveal>
      </PageTransition>,
    )
    expect(screen.getByRole("button", { name: "Continue" })).toBeEnabled()
    expect(screen.getByText("true")).toBeInTheDocument()
  })
})
