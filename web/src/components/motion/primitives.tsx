"use client"

import { useCallback, useEffect, useRef, useSyncExternalStore, type ComponentProps, type RefObject } from "react"

import { cn } from "@/lib/utils"

const preference = "(prefers-reduced-motion: reduce)"

function subscribe(notify: () => void) {
  const media = window.matchMedia?.(preference)
  media?.addEventListener("change", notify)
  return () => media?.removeEventListener("change", notify)
}

export function useReducedMotion() {
  return useSyncExternalStore(
    subscribe,
    () => window.matchMedia?.(preference).matches ?? true,
    () => true,
  )
}

function useVisibleMotion(ref: RefObject<HTMLElement | null>, start: () => (() => void) | undefined) {
  const reduced = useReducedMotion()
  useEffect(() => {
    const node = ref.current
    if (!node || reduced) return
    let stop: (() => void) | undefined
    if (typeof IntersectionObserver === "undefined") return start()
    const observer = new IntersectionObserver((entries) => {
      if (!entries.some((entry) => entry.isIntersecting)) return
      observer.disconnect()
      stop = start()
    })
    observer.observe(node)
    return () => {
      observer.disconnect()
      stop?.()
    }
  }, [ref, reduced, start])
}

export function Reveal({ className, variant = "slide", ...props }: ComponentProps<"div"> & { variant?: "fade" | "slide" }) {
  return <div className={cn(variant === "fade" ? "motion-fade" : "motion-slide", className)} {...props} />
}

export function Stagger({ className, ...props }: ComponentProps<"div">) {
  return <div className={cn("motion-stagger", className)} {...props} />
}

export function PageTransition({ style, ...props }: ComponentProps<"div">) {
  return <div style={{ ...style, viewTransitionName: "page-content" }} {...props} />
}

export function CountUp({ value, suffix = "", className }: { value: number; suffix?: string; className?: string }) {
  const ref = useRef<HTMLSpanElement>(null)
  const target = Math.round(value)
  const start = useCallback(() => {
    const node = ref.current
    if (!node) return
    let frame = 0
    let began: number | undefined
    const tick = (now: number) => {
      began ??= now
      const progress = Math.min((now - began) / 200, 1)
      node.textContent = `${Math.round(target * (1 - (1 - progress) ** 3))}${suffix}`
      if (progress < 1) frame = requestAnimationFrame(tick)
    }
    frame = requestAnimationFrame(tick)
    return () => {
      cancelAnimationFrame(frame)
      node.textContent = `${target}${suffix}`
    }
  }, [target, suffix])
  useVisibleMotion(ref, start)
  return (
    <span className={cn("inline-grid tabular-nums", className)}>
      <span className="sr-only">
        {target}
        {suffix}
      </span>
      <span aria-hidden className="invisible col-start-1 row-start-1">
        {target}
        {suffix}
      </span>
      <span ref={ref} aria-hidden className="col-start-1 row-start-1">
        {target}
        {suffix}
      </span>
    </span>
  )
}

export function AnimatedBar({ value, className }: { value: number; className?: string }) {
  const ref = useRef<HTMLDivElement>(null)
  const scale = Math.max(0, Math.min(value, 100)) / 100
  const start = useCallback(() => {
    const node = ref.current
    if (!node?.animate) return
    const animation = node.animate([{ transform: "scaleX(0)" }, { transform: `scaleX(${scale})` }], {
      duration: 200,
      easing: "cubic-bezier(0.2, 0, 0, 1)",
    })
    return () => animation.cancel()
  }, [scale])
  useVisibleMotion(ref, start)
  return <div ref={ref} aria-hidden className={cn("h-full w-full origin-left rounded-full", className)} style={{ transform: `scaleX(${scale})` }} />
}
