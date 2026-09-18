import React from 'react';

interface IconProps extends React.SVGProps<SVGSVGElement> {
  size?: number;
}

/**
 * Call-control icons drawn on the Radix Icons 15x15 grid (1.5px stroke) so
 * they sit visually alongside @radix-ui/react-icons.
 */
function Base({
  size = 18,
  children,
  ...props
}: IconProps): React.ReactElement {
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 15 15"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.5"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
      {...props}
    >
      {children}
    </svg>
  );
}

export function LogoMark({ size = 26, ...props }: IconProps): React.ReactElement {
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 26 26"
      fill="none"
      aria-hidden="true"
      {...props}
    >
      <rect x="1" y="1" width="24" height="24" rx="7" fill="currentColor" opacity="0.16" />
      <rect x="1" y="1" width="24" height="24" rx="7" stroke="currentColor" strokeWidth="1.5" />
      <path
        d="M8 10.2v3.6M11 7.8v6M14 10.2v3.6M17 8.6v4.4"
        stroke="currentColor"
        strokeWidth="1.7"
        strokeLinecap="round"
      />
      <circle cx="18.4" cy="8.2" r="1.6" fill="currentColor" stroke="none" />
    </svg>
  );
}

export function MicOnIcon(props: IconProps): React.ReactElement {
  return (
    <Base {...props}>
      <rect x="5.5" y="1.5" width="4" height="7" rx="2" />
      <path d="M3.5 6.5a3.5 3.5 0 0 0 7 0" />
      <path d="M7.5 10v2.5" />
      <path d="M5.5 13.5h4" />
    </Base>
  );
}

export function MicOffIcon(props: IconProps): React.ReactElement {
  return (
    <Base {...props}>
      <rect x="5.5" y="1.5" width="4" height="7" rx="2" />
      <path d="M3.5 6.5a3.5 3.5 0 0 0 7 0" />
      <path d="M7.5 10v2.5" />
      <path d="M5.5 13.5h4" />
      <path d="M2.5 2.5l10 10" />
    </Base>
  );
}

export function VideoOnIcon(props: IconProps): React.ReactElement {
  return (
    <Base {...props}>
      <rect x="1.5" y="4" width="8" height="7" rx="1.5" />
      <path d="M9.5 6.5l4-2.5v6l-4-2.5" />
    </Base>
  );
}

export function VideoOffIcon(props: IconProps): React.ReactElement {
  return (
    <Base {...props}>
      <rect x="1.5" y="4" width="8" height="7" rx="1.5" />
      <path d="M9.5 6.5l4-2.5v6l-4-2.5" />
      <path d="M2.5 2.5l10 10" />
    </Base>
  );
}

export function PhoneDownIcon(props: IconProps): React.ReactElement {
  return (
    <Base {...props} fill="currentColor" stroke="none">
      <path
        fillRule="evenodd"
        clipRule="evenodd"
        d="M2.6 3.1c.3-.3.8-.4 1.2-.2l1.3.7c.4.2.6.6.5 1.1l-.3 1.1c-.1.4 0 .8.3 1.1l1.9 1.9c.3.3.7.4 1.1.3l1.1-.3c.4-.1.9.1 1.1.5l.7 1.3c.2.4.1.9-.2 1.2l-.7.7c-.5.5-1.2.7-1.9.5-2.2-.6-4.3-2.7-4.9-4.9-.2-.7 0-1.4.5-1.9l.7-.7Z"
      />
    </Base>
  );
}
