"use client"

import { ArrowRight, ChevronDown } from "lucide-react"
import Link from "next/link"
import { useEffect, useRef, useState, type RefObject } from "react"

import { ButtonLink } from "@/components/ui/button-link"

import OrbitScene from "./orbit-scene"

type StoryStep = { title: string; body: string }

type LandingStoryProps = {
  eyebrow: string
  title: string
  titleAccent: string
  lead: string
  start: string
  foundAction: string
  signIn: string
  storyHint: string
  community: string
  steps: StoryStep[]
}

export default function LandingStory(props: LandingStoryProps) {
  const storyRef = useRef<HTMLElement>(null)
  const progress = useStoryProgress(storyRef)
  const activeIndex = progress < 0.28 ? 0 : progress < 0.62 ? 1 : 2
  const activeStep = props.steps[activeIndex] ?? props.steps[0]

  return (
    <section ref={storyRef} className="landing-story" aria-label="Perjalanan barang kembali">
      <div className="landing-story__stage">
        <OrbitScene storyRef={storyRef} />
        <div className="landing-story__wash" aria-hidden />
        <div className="landing-story__grid" aria-hidden />
        <div className="landing-story__inner">
          <div className="landing-story__hero-copy">
            <p className="landing-story__eyebrow">
              <span aria-hidden className="size-1.5 rounded-full bg-primary" />
              {props.eyebrow}
            </p>
            <h1 className="landing-story__title">
              {props.title}
              <br />
              <span>{props.titleAccent}</span>
            </h1>
            <p className="landing-story__lead">{props.lead}</p>
            <div className="landing-story__actions">
              <ButtonLink href="/daftar" size="lg">
                {props.start}
                <ArrowRight aria-hidden />
              </ButtonLink>
              <ButtonLink href="/lapor/temuan" size="lg" variant="outline" className="bg-card/90">
                {props.foundAction}
              </ButtonLink>
            </div>
            <Link href="/masuk" className="landing-story__sign-in">
              {props.signIn}
            </Link>
          </div>

          <div className="landing-story__chapter" aria-live="polite">
            <div className="landing-story__chapter-meta">
              <span>0{activeIndex + 1}</span>
              <span className="landing-story__chapter-line" aria-hidden />
              <span>03</span>
            </div>
            <h2 key={activeStep?.title} className="landing-story__chapter-title">
              {activeStep?.title}
            </h2>
            <p key={activeStep?.body} className="landing-story__chapter-body">
              {activeStep?.body}
            </p>
            <p className="landing-story__community">{props.community}</p>
          </div>

          <div className="landing-story__hint" aria-hidden>
            <span>{props.storyHint}</span>
            <ChevronDown className="size-4" />
          </div>
        </div>
      </div>
      <div className="landing-story__scroll-space" aria-hidden />
    </section>
  )
}

function useStoryProgress(storyRef: RefObject<HTMLElement | null>) {
  const [progress, setProgress] = useState(0)

  useEffect(() => {
    let frame = 0
    const update = () => {
      cancelAnimationFrame(frame)
      frame = requestAnimationFrame(() => {
        const node = storyRef.current
        if (!node) return
        const travel = Math.max(node.offsetHeight - window.innerHeight, 1)
        setProgress(clamp(-node.getBoundingClientRect().top / travel, 0, 1))
      })
    }
    update()
    window.addEventListener("scroll", update, { passive: true })
    window.addEventListener("resize", update)
    return () => {
      cancelAnimationFrame(frame)
      window.removeEventListener("scroll", update)
      window.removeEventListener("resize", update)
    }
  }, [storyRef])

  return progress
}

function clamp(value: number, min: number, max: number) {
  return Math.min(Math.max(value, min), max)
}
