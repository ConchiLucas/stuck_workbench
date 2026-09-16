export function PhraseIcon({ className = 'h-3.5 w-3.5' }: { className?: string }) {
  return (
    <svg
      viewBox="0 0 24 24"
      fill="currentColor"
      className={className}
      xmlns="http://www.w3.org/2000/svg"
      aria-hidden="true"
    >
      <path
        fillRule="evenodd"
        clipRule="evenodd"
        d="M4.5 3C3.12 3 2 4.12 2 5.5v9C2 15.88 3.12 17 4.5 17h1v3.25a.75.75 0 0 0 1.28.53L10.56 17H15.5c1.38 0 2.5-1.12 2.5-2.5v-9C18 4.12 16.88 3 15.5 3H4.5ZM7 8.5a1 1 0 0 1 1-1h6a1 1 0 1 1 0 2H8a1 1 0 0 1-1-1Zm1 3a1 1 0 1 0 0 2h4a1 1 0 1 0 0-2H8Z"
      />
      <path
        d="M19.5 7a.75.75 0 0 1 .75.75v7.25c0 .69-.56 1.25-1.25 1.25H18v1.5a.75.75 0 0 1-1.28.53L14.44 16H11a.75.75 0 0 1 0-1.5h3.75l2.25 2.25V15.5a.75.75 0 0 1 .75-.75h1.75V7.75A.75.75 0 0 1 19.5 7Z"
        opacity="0.65"
      />
    </svg>
  )
}
