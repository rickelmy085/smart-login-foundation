import { cn } from "@/lib/utils";

type BrandLogoProps = {
  className?: string;
  /** Use on dark/gradient backgrounds. */
  inverted?: boolean;
  size?: "sm" | "md" | "lg";
};

const sizes = {
  sm: { mark: "size-8 rounded-lg text-sm", text: "text-base" },
  md: { mark: "size-10 rounded-xl text-base", text: "text-lg" },
  lg: { mark: "size-12 rounded-2xl text-lg", text: "text-2xl" },
};

export function BrandLogo({ className, inverted = false, size = "md" }: BrandLogoProps) {
  const s = sizes[size];
  return (
    <div className={cn("flex items-center gap-3", className)} aria-label="B-Smart Copilot">
      <div
        aria-hidden="true"
        className={cn(
          "grid shrink-0 place-items-center font-display font-extrabold tracking-tight",
          "bg-brand text-brand-foreground shadow-elegant",
          s.mark,
        )}
      >
        B
      </div>
      <div className="flex flex-col leading-none">
        <span
          className={cn(
            "font-display font-bold tracking-tight",
            s.text,
            inverted ? "text-hero-foreground" : "text-foreground",
          )}
        >
          B-Smart <span className="text-brand">Copilot</span>
        </span>
        <span
          className={cn(
            "mt-1 text-[10px] font-semibold uppercase tracking-[0.2em]",
            inverted ? "text-hero-foreground/60" : "text-muted-foreground",
          )}
        >
          Assistente corporativo
        </span>
      </div>
    </div>
  );
}
