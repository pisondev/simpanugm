"use client"

import { Canvas, useFrame, useThree } from "@react-three/fiber"
import { PackageCheck } from "lucide-react"
import { useEffect, useMemo, useRef, useState, type RefObject } from "react"
import * as THREE from "three"

import { useReducedMotion } from "@/components/motion/primitives"

type ScrollTarget = { current: number }
type Point = [number, number, number]

export default function OrbitSceneCanvas({ storyRef }: { storyRef: RefObject<HTMLElement | null> }) {
  const reduced = useReducedMotion()
  const scroll = useScrollProgress(storyRef)
  const sceneRef = useRef<HTMLDivElement>(null)
  const [visible, setVisible] = useState(true)

  useEffect(() => {
    if (!sceneRef.current || typeof IntersectionObserver === "undefined") return
    const observer = new IntersectionObserver(([entry]) => setVisible(entry?.isIntersecting ?? false), { threshold: 0 })
    observer.observe(sceneRef.current)
    return () => observer.disconnect()
  }, [])

  return (
    <div ref={sceneRef} className="orbit-canvas" aria-hidden="true">
      <Canvas
        camera={{ position: [0, 0.4, 10], fov: 38 }}
        dpr={[1, 1.5]}
        fallback={<WebglFallback />}
        frameloop={reduced || !visible ? "never" : "always"}
        gl={{ alpha: true, antialias: true, powerPreference: "high-performance" }}
      >
        <ambientLight intensity={1.8} />
        <directionalLight position={[4, 6, 6]} intensity={3.6} color="#ffffff" />
        <directionalLight position={[-5, 2, 3]} intensity={1.4} color="#9dbdff" />
        <pointLight position={[0, 1.8, 2]} intensity={2.2} distance={7} color="#b8d0ff" />
        <LostAndFoundWorld reduced={reduced} scroll={scroll} />
      </Canvas>
    </div>
  )
}

function WebglFallback() {
  return (
    <div className="orbit-fallback" aria-hidden="true">
      <div className="orbit-fallback__ring orbit-fallback__ring--outer" />
      <div className="orbit-fallback__ring orbit-fallback__ring--inner" />
      <span className="orbit-fallback__parcel">
        <PackageCheck aria-hidden />
      </span>
    </div>
  )
}

function LostAndFoundWorld({ reduced, scroll }: { reduced: boolean; scroll: ScrollTarget }) {
  const { camera, size } = useThree()
  const root = useRef<THREE.Group>(null)
  const lostItems = useRef<THREE.Group>(null)
  const network = useRef<THREE.Group>(null)
  const verification = useRef<THREE.Group>(null)
  const returnRoute = useRef<THREE.Group>(null)
  const parcel = useRef<THREE.Group>(null)
  const progress = useRef(0)
  const cameraPosition = useMemo(() => new THREE.Vector3(), [])
  const cameraLookAt = useMemo(() => new THREE.Vector3(), [])

  useFrame(({ clock, pointer }, delta) => {
    if (!root.current || !lostItems.current || !network.current || !verification.current || !returnRoute.current || !parcel.current) return

    const target = reduced ? 0 : scroll.current
    progress.current = THREE.MathUtils.damp(progress.current, target, 4.3, delta)
    const p = smoothstep(progress.current)
    const time = reduced ? 0 : clock.elapsedTime
    const narrow = size.width < 720

    cameraPosition.set((narrow ? 0 : 0.85) - p * (narrow ? 0.1 : 1.45), 0.55 + p * 0.7 + Math.sin(time * 0.35) * 0.035, 10.1 - p * 3.2)
    camera.position.lerp(cameraPosition, 1 - Math.exp(-delta * 3.2))
    cameraLookAt.set((narrow ? 0 : 0.25) + Math.sin(p * Math.PI) * 0.45, 0.05 + p * 0.15, -0.15)
    camera.lookAt(cameraLookAt)

    root.current.position.x = narrow ? 0 : 0.65
    root.current.rotation.x = THREE.MathUtils.damp(root.current.rotation.x, pointer.y * 0.035, 3, delta)
    root.current.rotation.y = THREE.MathUtils.damp(root.current.rotation.y, pointer.x * 0.055 - p * 0.08, 3, delta)
    root.current.position.y = THREE.MathUtils.damp(root.current.position.y, Math.sin(time * 0.45) * 0.035, 3, delta)

    const itemsOut = smoothstep(p / 0.34)
    const networkIn = smoothstep((p - 0.12) / 0.34)
    const verifyIn = smoothstep((p - 0.43) / 0.34)
    const returnIn = smoothstep((p - 0.7) / 0.3)
    lostItems.current.scale.setScalar(1 - itemsOut * 0.72)
    lostItems.current.position.y = itemsOut * 0.5
    network.current.scale.setScalar(0.12 + networkIn * 0.88)
    network.current.position.y = (1 - networkIn) * 0.25
    verification.current.scale.setScalar(0.08 + verifyIn * 0.92)
    verification.current.position.y = (1 - verifyIn) * -0.2
    returnRoute.current.scale.setScalar(0.08 + returnIn * 0.92)
    returnRoute.current.position.y = (1 - returnIn) * -0.4

    parcel.current.rotation.y = THREE.MathUtils.damp(parcel.current.rotation.y, p * 0.36, 3.5, delta)
    parcel.current.rotation.z = THREE.MathUtils.damp(parcel.current.rotation.z, Math.sin(time * 0.5) * 0.035, 3, delta)
    parcel.current.position.y = THREE.MathUtils.damp(parcel.current.position.y, Math.sin(time * 0.8) * 0.06, 3, delta)
  })

  return (
    <group ref={root}>
      <WorldFloor />
      <CampusBackdrop />
      <group ref={lostItems}>
        <group position={[-2.05, 1.05, 0.25]} rotation={[0.2, -0.2, -0.18]}>
          <WalletObject />
        </group>
        <group position={[2.02, 1.1, 0.1]} rotation={[-0.14, 0.35, 0.16]}>
          <KeyObject />
        </group>
        <group position={[-1.9, -1.0, -0.2]} rotation={[0.18, -0.25, 0.1]}>
          <BackpackObject />
        </group>
        <group position={[1.9, -0.95, 0.15]} rotation={[-0.12, 0.25, -0.12]}>
          <EarbudsObject />
        </group>
      </group>

      <group ref={network}>
        <NetworkCore />
        <NetworkLine
          points={[
            [-2.05, 1.05, 0.2],
            [0, 0.05, 0.2],
            [2.02, 1.1, 0.05],
          ]}
          color="#7ea6f7"
        />
        <NetworkLine
          points={[
            [-1.9, -1, -0.15],
            [0, 0.05, 0.2],
            [1.9, -0.95, 0.12],
          ]}
          color="#a6bdf1"
        />
        <OrbitNode position={[-2.05, 1.05, 0.2]} color="#1e40af" />
        <OrbitNode position={[2.02, 1.1, 0.05]} color="#b45309" />
        <OrbitNode position={[-1.9, -1, -0.15]} color="#15803d" />
        <OrbitNode position={[1.9, -0.95, 0.12]} color="#7c3aed" />
      </group>

      <group ref={verification}>
        <ScannerGate />
        <Beam start={[-2.3, -1.75, 0]} end={[2.3, -1.75, 0]} color="#8db0f4" radius={0.012} />
        <VerificationBeacon position={[0, -1.68, 0]} />
      </group>

      <group ref={returnRoute}>
        <group ref={parcel} position={[0, 0.05, 0.45]}>
          <Parcel />
        </group>
        <ReturnPad position={[-1.35, -1.7, 0]} color="#dbeafe" />
        <ReturnPad position={[1.35, -1.7, 0]} color="#dcfce7" />
        <NetworkLine
          points={[
            [-1.35, -1.7, 0],
            [0, -0.1, 0.35],
            [1.35, -1.7, 0],
          ]}
          color="#78a8f8"
        />
        <SuccessBadge position={[0, 1.45, 0.3]} />
      </group>
    </group>
  )
}

function WorldFloor() {
  return (
    <group position={[0, -2.35, -1.3]}>
      <mesh rotation={[-Math.PI / 2, 0, 0]} receiveShadow>
        <planeGeometry args={[20, 15]} />
        <meshStandardMaterial color="#edf3ff" roughness={0.92} metalness={0} />
      </mesh>
      <gridHelper args={[20, 20, "#b5c8ed", "#e4ecfb"]} position={[0, 0.015, 0]} />
      <mesh position={[0, 0.04, 0]} rotation={[-Math.PI / 2, 0, 0]}>
        <ringGeometry args={[4.45, 4.48, 64]} />
        <meshBasicMaterial color="#a7bdf0" transparent opacity={0.55} />
      </mesh>
    </group>
  )
}

function CampusBackdrop() {
  return (
    <group position={[0, -1.35, -2.6]}>
      <CampusBlock position={[-3.6, 0, -0.6]} scale={[1.2, 2.4, 1.4]} color="#dbe7fb" />
      <CampusBlock position={[3.7, 0.35, -1]} scale={[1.45, 3.1, 1.55]} color="#e4edf9" />
      <CampusBlock position={[-1.35, -0.15, -1.6]} scale={[1.05, 1.45, 0.85]} color="#cbdcf8" />
      <CampusBlock position={[1.35, -0.05, -1.7]} scale={[1.2, 1.75, 0.9]} color="#d7e6f7" />
      <mesh position={[0, 0.25, -1.9]}>
        <cylinderGeometry args={[1.7, 1.95, 0.24, 48]} />
        <meshStandardMaterial color="#f8fbff" roughness={0.68} />
      </mesh>
    </group>
  )
}

function CampusBlock({ position, scale, color }: { position: Point; scale: Point; color: string }) {
  return (
    <group position={position}>
      <mesh scale={scale} castShadow>
        <boxGeometry args={[1, 1, 1]} />
        <meshStandardMaterial color={color} roughness={0.62} />
      </mesh>
      <mesh position={[0, scale[1] * 0.56, 0]} rotation={[0, Math.PI / 4, 0]} scale={[scale[0] * 0.88, 0.16, scale[2] * 0.88]}>
        <boxGeometry args={[1, 1, 1]} />
        <meshStandardMaterial color="#b8cdf1" roughness={0.68} />
      </mesh>
      <mesh position={[0, 0, scale[2] * 0.51]} scale={[scale[0] * 0.46, scale[1] * 0.16, 0.025]}>
        <boxGeometry args={[1, 1, 1]} />
        <meshBasicMaterial color="#ffffff" transparent opacity={0.5} />
      </mesh>
    </group>
  )
}

function NetworkCore() {
  return (
    <group position={[0, 0.05, 0.25]}>
      <mesh>
        <sphereGeometry args={[0.28, 24, 16]} />
        <meshStandardMaterial color="#2454bd" roughness={0.22} metalness={0.18} emissive="#1e40af" emissiveIntensity={0.12} />
      </mesh>
      <mesh rotation={[Math.PI / 2, 0, -0.18]}>
        <torusGeometry args={[0.54, 0.018, 8, 64]} />
        <meshBasicMaterial color="#75a2f5" transparent opacity={0.78} />
      </mesh>
      <mesh rotation={[0.3, 0.1, 0.6]}>
        <torusGeometry args={[0.8, 0.012, 8, 64]} />
        <meshBasicMaterial color="#a9c2f4" transparent opacity={0.64} />
      </mesh>
    </group>
  )
}

function ScannerGate() {
  return (
    <group position={[0, -0.65, 0.35]}>
      <mesh position={[-1.72, 0, 0]}>
        <boxGeometry args={[0.12, 2.1, 0.18]} />
        <meshStandardMaterial color="#87a8e8" roughness={0.35} metalness={0.18} />
      </mesh>
      <mesh position={[1.72, 0, 0]}>
        <boxGeometry args={[0.12, 2.1, 0.18]} />
        <meshStandardMaterial color="#87a8e8" roughness={0.35} metalness={0.18} />
      </mesh>
      <mesh position={[0, 1.03, 0]}>
        <boxGeometry args={[3.55, 0.12, 0.18]} />
        <meshStandardMaterial color="#668bd8" roughness={0.32} metalness={0.2} />
      </mesh>
      <mesh position={[0, 0.25, 0.12]} rotation={[0, 0, Math.PI / 2]}>
        <torusGeometry args={[0.72, 0.025, 8, 64, Math.PI]} />
        <meshBasicMaterial color="#bad0fa" transparent opacity={0.8} />
      </mesh>
    </group>
  )
}

function VerificationBeacon({ position }: { position: Point }) {
  return (
    <group position={position}>
      <mesh>
        <cylinderGeometry args={[0.2, 0.25, 0.38, 20]} />
        <meshStandardMaterial color="#f3f7ff" roughness={0.3} />
      </mesh>
      <mesh position={[0, 0.25, 0]}>
        <sphereGeometry args={[0.12, 16, 12]} />
        <meshStandardMaterial color="#15803d" emissive="#15803d" emissiveIntensity={0.18} />
      </mesh>
    </group>
  )
}

function ReturnPad({ position, color }: { position: Point; color: string }) {
  return (
    <group position={position}>
      <mesh rotation={[-Math.PI / 2, 0, 0]}>
        <cylinderGeometry args={[0.65, 0.72, 0.12, 32]} />
        <meshStandardMaterial color={color} roughness={0.48} />
      </mesh>
      <mesh position={[0, 0.08, 0]} rotation={[-Math.PI / 2, 0, 0]}>
        <torusGeometry args={[0.5, 0.02, 8, 48]} />
        <meshBasicMaterial color="#83a7ed" transparent opacity={0.62} />
      </mesh>
    </group>
  )
}

function SuccessBadge({ position }: { position: Point }) {
  return (
    <group position={position} rotation={[0.1, 0.1, 0]}>
      <mesh>
        <cylinderGeometry args={[0.44, 0.44, 0.12, 32]} />
        <meshStandardMaterial color="#15803d" roughness={0.35} metalness={0.08} />
      </mesh>
      <mesh position={[0, 0.12, 0]} rotation={[0, 0, -0.6]}>
        <boxGeometry args={[0.1, 0.4, 0.08]} />
        <meshStandardMaterial color="#ffffff" roughness={0.3} />
      </mesh>
      <mesh position={[0.12, 0.22, 0]} rotation={[0, 0, 0.7]}>
        <boxGeometry args={[0.1, 0.62, 0.08]} />
        <meshStandardMaterial color="#ffffff" roughness={0.3} />
      </mesh>
    </group>
  )
}

function NetworkLine({ points, color }: { points: Point[]; color: string }) {
  const geometry = useMemo(() => {
    const curve = new THREE.CatmullRomCurve3(points.map(([x, y, z]) => new THREE.Vector3(x, y, z)))
    return new THREE.TubeGeometry(curve, 32, 0.014, 6, false)
  }, [points])
  return (
    <mesh geometry={geometry}>
      <meshBasicMaterial color={color} transparent opacity={0.65} />
    </mesh>
  )
}

function Beam({ start, end, color, radius }: { start: Point; end: Point; color: string; radius: number }) {
  const startVector = useMemo(() => new THREE.Vector3(...start), [start])
  const endVector = useMemo(() => new THREE.Vector3(...end), [end])
  const direction = useMemo(() => endVector.clone().sub(startVector), [endVector, startVector])
  const midpoint = useMemo(() => startVector.clone().add(endVector).multiplyScalar(0.5), [endVector, startVector])
  const quaternion = useMemo(() => new THREE.Quaternion().setFromUnitVectors(new THREE.Vector3(0, 1, 0), direction.clone().normalize()), [direction])
  return (
    <mesh position={midpoint} quaternion={quaternion}>
      <cylinderGeometry args={[radius, radius, direction.length(), 8]} />
      <meshBasicMaterial color={color} transparent opacity={0.58} />
    </mesh>
  )
}

function Parcel() {
  return (
    <group>
      <mesh castShadow>
        <boxGeometry args={[1.35, 1.05, 1.12]} />
        <meshStandardMaterial color="#2454bd" roughness={0.48} metalness={0.04} />
      </mesh>
      <mesh position={[0, 0.02, 0.575]}>
        <boxGeometry args={[0.82, 0.56, 0.025]} />
        <meshStandardMaterial color="#dbeafe" roughness={0.35} />
      </mesh>
      <mesh position={[0, 0.57, 0]}>
        <boxGeometry args={[0.18, 0.07, 1.17]} />
        <meshStandardMaterial color="#3b65c6" roughness={0.4} />
      </mesh>
      <mesh position={[0, 0.58, 0]}>
        <boxGeometry args={[0.18, 0.025, 0.62]} />
        <meshStandardMaterial color="#dbeafe" roughness={0.35} />
      </mesh>
      <group position={[0, 0.02, 0.595]}>
        <mesh position={[-0.12, 0.02, 0]} rotation={[0, 0, -0.72]}>
          <boxGeometry args={[0.08, 0.28, 0.04]} />
          <meshStandardMaterial color="#15803d" roughness={0.35} />
        </mesh>
        <mesh position={[0.07, 0.09, 0]} rotation={[0, 0, 0.72]}>
          <boxGeometry args={[0.08, 0.46, 0.04]} />
          <meshStandardMaterial color="#15803d" roughness={0.35} />
        </mesh>
      </group>
    </group>
  )
}

function WalletObject() {
  return (
    <group>
      <mesh castShadow>
        <boxGeometry args={[0.72, 0.16, 0.98]} />
        <meshStandardMaterial color="#31445f" roughness={0.38} />
      </mesh>
      <mesh position={[0, 0.095, 0.08]}>
        <boxGeometry args={[0.55, 0.025, 0.04]} />
        <meshStandardMaterial color="#88a8e8" roughness={0.3} />
      </mesh>
      <mesh position={[0, 0.095, -0.18]}>
        <boxGeometry args={[0.4, 0.025, 0.04]} />
        <meshStandardMaterial color="#88a8e8" roughness={0.3} />
      </mesh>
    </group>
  )
}

function KeyObject() {
  return (
    <group>
      <mesh position={[0.15, 0, 0]} castShadow>
        <boxGeometry args={[0.75, 0.07, 0.11]} />
        <meshStandardMaterial color="#d7a642" roughness={0.28} metalness={0.5} />
      </mesh>
      <mesh position={[-0.32, 0, 0]} rotation={[Math.PI / 2, 0, 0]}>
        <torusGeometry args={[0.22, 0.055, 10, 32]} />
        <meshStandardMaterial color="#d7a642" roughness={0.28} metalness={0.5} />
      </mesh>
      <mesh position={[0.48, 0, 0]}>
        <boxGeometry args={[0.1, 0.18, 0.11]} />
        <meshStandardMaterial color="#d7a642" roughness={0.28} metalness={0.5} />
      </mesh>
    </group>
  )
}

function BackpackObject() {
  return (
    <group>
      <mesh castShadow>
        <boxGeometry args={[0.78, 0.92, 0.4]} />
        <meshStandardMaterial color="#3b7f70" roughness={0.6} />
      </mesh>
      <mesh position={[0, -0.12, 0.215]}>
        <boxGeometry args={[0.52, 0.28, 0.08]} />
        <meshStandardMaterial color="#a4d3c7" roughness={0.55} />
      </mesh>
      <mesh position={[0, 0.57, 0]} rotation={[0, 0, Math.PI / 2]}>
        <torusGeometry args={[0.22, 0.055, 8, 24, Math.PI]} />
        <meshStandardMaterial color="#a4d3c7" roughness={0.55} />
      </mesh>
    </group>
  )
}

function EarbudsObject() {
  return (
    <group>
      <mesh castShadow>
        <boxGeometry args={[0.58, 0.28, 0.45]} />
        <meshStandardMaterial color="#f4f7fb" roughness={0.25} />
      </mesh>
      <mesh position={[-0.14, 0.2, 0]}>
        <sphereGeometry args={[0.12, 16, 12]} />
        <meshStandardMaterial color="#ffffff" roughness={0.22} />
      </mesh>
      <mesh position={[0.14, 0.2, 0]}>
        <sphereGeometry args={[0.12, 16, 12]} />
        <meshStandardMaterial color="#ffffff" roughness={0.22} />
      </mesh>
    </group>
  )
}

function OrbitNode({ position, color }: { position: Point; color: string }) {
  return (
    <mesh position={position}>
      <sphereGeometry args={[0.075, 16, 12]} />
      <meshBasicMaterial color={color} transparent opacity={0.9} />
    </mesh>
  )
}

function useScrollProgress(storyRef: RefObject<HTMLElement | null>): ScrollTarget {
  const target = useRef(0)

  useEffect(() => {
    let frame = 0
    const update = () => {
      cancelAnimationFrame(frame)
      frame = requestAnimationFrame(() => {
        const node = storyRef.current
        if (!node) return
        const range = Math.max(node.offsetHeight - window.innerHeight, 1)
        target.current = THREE.MathUtils.clamp(-node.getBoundingClientRect().top / range, 0, 1)
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

  return target
}

function smoothstep(value: number) {
  const clamped = THREE.MathUtils.clamp(value, 0, 1)
  return clamped * clamped * (3 - 2 * clamped)
}
