interface CategoryIconProps {
  icon: string
  name?: string
  size?: 'sm' | 'md' | 'lg'
  className?: string
}

const sizeMap = {
  sm: 'w-7 h-7 text-sm',
  md: 'w-9 h-9 text-base',
  lg: 'w-12 h-12 text-xl',
}

export default function CategoryIcon({ icon, name, size = 'md', className = '' }: CategoryIconProps) {
  return (
    <div
      className={`${sizeMap[size]} rounded-full bg-blue-50 flex items-center justify-center flex-shrink-0 ${className}`}
      title={name}
    >
      <span role="img" aria-label={name}>
        {icon || '📌'}
      </span>
    </div>
  )
}
