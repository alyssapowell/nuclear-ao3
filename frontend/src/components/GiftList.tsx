'use client'

import { useEffect, useState } from 'react'
import { Gift as GiftIcon } from 'lucide-react'
import { getWorkGifts, Gift } from '@/lib/api'

interface GiftListProps {
  workId: string;
  gifts?: Gift[];
  authToken?: string;
}

export function GiftList({ workId, gifts: initialGifts, authToken }: GiftListProps) {
  const [gifts, setGifts] = useState<Gift[]>(initialGifts || [])
  const [isLoading, setIsLoading] = useState(!initialGifts)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!initialGifts) {
      fetchGifts()
    }
  }, [workId, initialGifts])

  const fetchGifts = async () => {
    try {
      setIsLoading(true)
      setError(null)
      const result = await getWorkGifts(workId, authToken)
      setGifts(result.gifts || [])
    } catch (error) {
      setError(error instanceof Error ? error.message : 'Failed to load gifts')
    } finally {
      setIsLoading(false)
    }
  }

  if (isLoading) {
    return (
      <div className="flex items-center gap-2 text-sm text-gray-600">
        <GiftIcon className="h-4 w-4 animate-pulse" />
        <span>Loading gifts...</span>
      </div>
    )
  }

  if (error) {
    return (
      <div className="flex items-center gap-2 text-sm text-red-600">
        <GiftIcon className="h-4 w-4" />
        <span>Error loading gifts</span>
      </div>
    )
  }

  if (!gifts || gifts.length === 0) {
    return null
  }

  return (
    <div className="flex items-center gap-2 text-sm text-gray-700">
      <GiftIcon className="h-4 w-4 text-purple-600" />
      <span>
        Gifted to: {gifts.map((gift, index) => (
          <span key={gift.id}>
            {gift.recipient?.username ? (
              <span className="font-medium text-purple-700">
                {gift.recipient.username}
              </span>
            ) : (
              <span className="font-medium text-purple-700">
                {gift.recipient_name}
              </span>
            )}
            {index < gifts.length - 1 && ', '}
          </span>
        ))}
      </span>
    </div>
  )
}

// Hook to refresh gifts (for parent components)
export function useGiftList(workId: string, authToken?: string) {
  const [gifts, setGifts] = useState<Gift[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const fetchGifts = async () => {
    try {
      setIsLoading(true)
      setError(null)
      const result = await getWorkGifts(workId, authToken)
      setGifts(result.gifts || [])
    } catch (error) {
      setError(error instanceof Error ? error.message : 'Failed to load gifts')
    } finally {
      setIsLoading(false)
    }
  }

  useEffect(() => {
    fetchGifts()
  }, [workId])

  return {
    gifts,
    isLoading,
    error,
    refetch: fetchGifts
  }
}