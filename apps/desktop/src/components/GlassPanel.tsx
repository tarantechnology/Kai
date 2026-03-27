import { forwardRef } from "react";
import type { PropsWithChildren } from "react";

interface GlassPanelProps extends PropsWithChildren {
  className?: string;
}

export const GlassPanel = forwardRef<HTMLDivElement, GlassPanelProps>(({ className = "", children }, ref) => (
  <div ref={ref} className={`glass-panel ${className}`.trim()}>
    {children}
  </div>
));

GlassPanel.displayName = "GlassPanel";
