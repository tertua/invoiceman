import { useEffect, useRef, useState } from "react";
import { animate, useMotionValue, useReducedMotion } from "framer-motion";

// Counts a numeric StatCard value up from 0 on mount / value change.
//
// The caller owns formatting via `render(n) -> string`, so a money value animates
// through *valid* formatted strings (each frame is re-formatted) instead of a
// half-parsed float. Reduced-motion is respected per-component: MotionConfig in
// AppShell only wraps page transitions, not StatCards.
export function CountUpValue({ value, render, className }) {
  const reduce = useReducedMotion();
  const mv = useMotionValue(0);
  const [text, setText] = useState(() => render(value));
  const renderRef = useRef(render);

  useEffect(() => {
    renderRef.current = render;
  }, [render]);

  useEffect(() => {
    if (reduce) {
      mv.set(value);
      setText(renderRef.current(value));
      return undefined;
    }
    const controls = animate(mv, value, {
      duration: 0.6,
      ease: [0.16, 1, 0.3, 1],
      onUpdate: (latest) => setText(renderRef.current(latest)),
      onComplete: () => setText(renderRef.current(value)),
    });
    return () => controls.stop();
  }, [value, reduce, mv]);

  return <span className={className}>{text}</span>;
}
