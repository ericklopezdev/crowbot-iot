import type { ButtonHTMLAttributes, ReactNode } from "react";
import "./ui.css";

export function Card({
  title,
  children,
  style,
}: {
  title?: string;
  children: ReactNode;
  style?: React.CSSProperties;
}) {
  return (
    <section className="card" style={style}>
      {title && <div className="card-title">{title}</div>}
      {children}
    </section>
  );
}

interface BtnProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: "default" | "primary";
}
export function Button({ variant = "default", className = "", ...rest }: BtnProps) {
  const v = variant === "primary" ? "btn-primary" : "";
  return <button className={`btn ${v} ${className}`} {...rest} />;
}

export function Badge({ label, count }: { label: string; count?: number }) {
  return (
    <span className="badge">
      {label}
      {count !== undefined && <span className="badge-count">{count}</span>}
    </span>
  );
}

export function Spinner() {
  return (
    <div className="center">
      <div className="spinner" />
    </div>
  );
}

export function EmptyState({ message = "Sin datos todavía" }: { message?: string }) {
  return <div className="empty">{message}</div>;
}

// Render helper: spinner / error / empty / content.
export function Async<T>({
  loading,
  error,
  data,
  isEmpty,
  children,
}: {
  loading: boolean;
  error: string | null;
  data: T | null;
  isEmpty?: (d: T) => boolean;
  children: (d: T) => ReactNode;
}) {
  if (loading) return <Spinner />;
  if (error) return <EmptyState message={error} />;
  if (!data || (isEmpty && isEmpty(data))) return <EmptyState />;
  return <>{children(data)}</>;
}
