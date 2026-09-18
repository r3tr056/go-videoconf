import React, { useState, useRef, useEffect } from 'react';
import * as Label from '@radix-ui/react-label';
import { Slot } from '@radix-ui/react-slot';
import * as Tooltip from '@radix-ui/react-tooltip';
import { CheckIcon, CopyIcon, ExclamationTriangleIcon } from '@radix-ui/react-icons';

/* ---------- Button ---------- */

type ButtonVariant = 'primary' | 'teal' | 'ghost' | 'danger';
type ButtonSize = 'sm' | 'md' | 'lg';

interface ButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: ButtonVariant;
  size?: ButtonSize;
  block?: boolean;
  loading?: boolean;
  asChild?: boolean;
}

export const Button = React.forwardRef<HTMLButtonElement, ButtonProps>(
  (
    {
      variant = 'primary',
      size = 'md',
      block = false,
      loading = false,
      asChild = false,
      className = '',
      disabled,
      children,
      ...props
    },
    ref,
  ) => {
    const Comp = asChild ? Slot : 'button';
    const cls = [
      'vc-btn',
      `vc-btn-${variant}`,
      size !== 'md' ? `vc-btn-${size}` : '',
      block ? 'vc-btn-block' : '',
      className,
    ]
      .filter(Boolean)
      .join(' ');
    return (
      <Comp ref={ref} className={cls} disabled={disabled || loading} {...props}>
        {loading && <span className="vc-spinner" aria-hidden="true" />}
        {children}
      </Comp>
    );
  },
);
Button.displayName = 'Button';

/* ---------- IconButton ---------- */

interface IconButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  label: string;
  bordered?: boolean;
}

export const IconButton = React.forwardRef<HTMLButtonElement, IconButtonProps>(
  ({ label, bordered = false, className = '', children, ...props }, ref) => (
    <button
      ref={ref}
      type="button"
      aria-label={label}
      title={label}
      className={`vc-icon-btn${bordered ? ' bordered' : ''} ${className}`.trim()}
      {...props}
    >
      {children}
    </button>
  ),
);
IconButton.displayName = 'IconButton';

/* ---------- Field ---------- */

interface FieldProps extends React.InputHTMLAttributes<HTMLInputElement> {
  label: string;
  id: string;
  mono?: boolean;
  trailing?: React.ReactNode;
}

export function Field({
  label,
  id,
  mono = false,
  trailing,
  className = '',
  ...props
}: FieldProps): React.ReactElement {
  return (
    <div className="vc-field">
      <Label.Root className="vc-label" htmlFor={id}>
        {label}
      </Label.Root>
      <div className="vc-input-wrap">
        <input
          id={id}
          className={`vc-input${mono ? ' vc-input-mono' : ''}${trailing ? ' has-trailing' : ''} ${className}`.trim()}
          {...props}
        />
        {trailing && <span className="vc-input-trailing">{trailing}</span>}
      </div>
    </div>
  );
}

/* ---------- Callout ---------- */

interface CalloutProps {
  tone?: 'error' | 'info';
  role?: 'alert' | 'status';
  children: React.ReactNode;
}

export function Callout({
  tone = 'error',
  role = 'alert',
  children,
}: CalloutProps): React.ReactElement {
  return (
    <div className={`vc-callout vc-callout-${tone}`} role={role}>
      <ExclamationTriangleIcon width={16} height={16} aria-hidden="true" />
      <div>{children}</div>
    </div>
  );
}

/* ---------- Badge ---------- */

interface BadgeProps {
  live?: boolean;
  children: React.ReactNode;
}

export function Badge({ live = false, children }: BadgeProps): React.ReactElement {
  return (
    <span className={`vc-badge${live ? ' vc-badge-live' : ''}`}>
      <span className="vc-dot" aria-hidden="true" />
      {children}
    </span>
  );
}

/* ---------- CopyButton ---------- */

export async function copyText(value: string): Promise<boolean> {
  try {
    if (navigator.clipboard?.writeText) {
      await navigator.clipboard.writeText(value);
      return true;
    }
  } catch {
    // Fall through to the legacy path below.
  }
  try {
    const area = document.createElement('textarea');
    area.value = value;
    area.setAttribute('readonly', '');
    area.style.position = 'absolute';
    area.style.left = '-9999px';
    document.body.appendChild(area);
    area.select();
    const ok = document.execCommand('copy');
    document.body.removeChild(area);
    return ok;
  } catch {
    return false;
  }
}

interface CopyButtonProps {
  value: string;
  label: string;
  size?: number;
}

export function CopyButton({
  value,
  label,
  size = 15,
}: CopyButtonProps): React.ReactElement {
  const [copied, setCopied] = useState(false);
  const timer = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(
    () => () => {
      if (timer.current) clearTimeout(timer.current);
    },
    [],
  );

  const onClick = async () => {
    if (await copyText(value)) {
      setCopied(true);
      if (timer.current) clearTimeout(timer.current);
      timer.current = setTimeout(() => setCopied(false), 1600);
    }
  };

  return (
    <IconButton label={copied ? 'Copied' : label} onClick={onClick}>
      {copied ? (
        <CheckIcon width={size} height={size} aria-hidden="true" />
      ) : (
        <CopyIcon width={size} height={size} aria-hidden="true" />
      )}
    </IconButton>
  );
}

/* ---------- Tooltip shorthand ---------- */

interface TipProps {
  content: string;
  children: React.ReactElement;
  side?: 'top' | 'bottom' | 'left' | 'right';
}

export function Tip({ content, children, side = 'top' }: TipProps): React.ReactElement {
  return (
    <Tooltip.Root delayDuration={250}>
      <Tooltip.Trigger asChild>{children}</Tooltip.Trigger>
      <Tooltip.Portal>
        <Tooltip.Content className="vc-tooltip" side={side} sideOffset={6}>
          {content}
          <Tooltip.Arrow className="vc-tooltip-arrow" />
        </Tooltip.Content>
      </Tooltip.Portal>
    </Tooltip.Root>
  );
}
