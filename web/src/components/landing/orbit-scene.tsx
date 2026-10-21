"use client"

import dynamic from "next/dynamic"
import { Backpack, KeyRound, PackageCheck, Wallet } from "lucide-react"
import type { RefObject } from "react"

const OrbitCanvas = dynamic(() => import("./orbit-scene-canvas"), {
  ssr: false,
  loading: () => <OrbitFallback />,
})

export default function OrbitScene({ storyRef }: { storyRef: RefObject<HTMLElement | null> }) {
  return <OrbitCanvas storyRef={storyRef} />
}

function OrbitFallback() {
  return (
    <div className="orbit-fallback" aria-hidden="true">
      <div className="orbit-fallback__ring orbit-fallback__ring--outer" />
      <div className="orbit-fallback__ring orbit-fallback__ring--inner" />
      <span className="orbit-fallback__item orbit-fallback__item--wallet">
        <Wallet aria-hidden />
      </span>
      <span className="orbit-fallback__item orbit-fallback__item--key">
        <KeyRound aria-hidden />
      </span>
      <span className="orbit-fallback__item orbit-fallback__item--bag">
        <Backpack aria-hidden />
      </span>
      <span className="orbit-fallback__parcel">
        <PackageCheck aria-hidden />
      </span>
    </div>
  )
}
