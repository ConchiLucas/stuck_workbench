import { CaretLeft, CaretRight, BookOpen, MapTrifold, SpeakerHigh, Star, X } from '@phosphor-icons/react'

const weight = 'light' as const

export const BrandIcon = () => <BookOpen size={22} weight="thin" aria-hidden="true" />
export const ListenIcon = ({ size = 28 }: { size?: number }) => <SpeakerHigh size={size} weight={weight} aria-hidden="true" />
export const MapIcon = () => <MapTrifold size={20} weight={weight} aria-hidden="true" />
export const ExitIcon = () => <X size={20} weight="regular" aria-hidden="true" />
export const PrevIcon = () => <CaretLeft size={22} weight="regular" aria-hidden="true" />
export const NextIcon = () => <CaretRight size={22} weight="regular" aria-hidden="true" />
export const StarIcon = ({ lit }: { lit: boolean }) => <Star size={36} weight={lit ? 'fill' : 'light'} aria-hidden="true" />
