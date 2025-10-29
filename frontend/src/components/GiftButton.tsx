'use client'

import { useState } from 'react'
import { useAuth } from '@/lib/auth'
import { giftWork, CreateGiftRequest } from '@/lib/api'
import { Button } from '@/components/ui/Button'
import { 
  Dialog, 
  DialogContent, 
  DialogDescription, 
  DialogHeader, 
  DialogTitle, 
  DialogTrigger 
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Gift } from 'lucide-react'

interface GiftButtonProps {
  workId: string;
  isAuthor: boolean;
  onGiftCreated?: (gift: any) => void;
}

export function GiftButton({ workId, isAuthor, onGiftCreated }: GiftButtonProps) {
  const { user, token } = useAuth()
  const [isOpen, setIsOpen] = useState(false)
  const [isLoading, setIsLoading] = useState(false)
  const [recipientName, setRecipientName] = useState('')
  const [error, setError] = useState<string | null>(null)

  if (!user || !isAuthor) {
    return null
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    
    if (!recipientName.trim()) {
      setError('Recipient name is required')
      return
    }

    setIsLoading(true)
    setError(null)

    try {
      const giftData: CreateGiftRequest = {
        recipient_name: recipientName.trim()
      }

      const result = await giftWork(workId, giftData, token || undefined)
      
      onGiftCreated?.(result.gift)
      setIsOpen(false)
      setRecipientName('')
    } catch (error) {
      setError(error instanceof Error ? error.message : 'Failed to create gift')
    } finally {
      setIsLoading(false)
    }
  }

  return (
    <Dialog open={isOpen} onOpenChange={setIsOpen}>
      <DialogTrigger asChild>
        <Button variant="outline" size="sm" className="gap-1">
          <Gift className="h-4 w-4" />
          Gift
        </Button>
      </DialogTrigger>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Gift this Work</DialogTitle>
          <DialogDescription>
            Add a recipient to gift this work to someone special. This will display 
            their name on the work as a gift recipient.
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="recipient">Recipient Name</Label>
            <Input
              id="recipient"
              type="text"
              placeholder="Enter recipient name..."
              value={recipientName}
              onChange={(e) => setRecipientName(e.target.value)}
              required
              disabled={isLoading}
            />
          </div>
          
          {error && (
            <div className="text-sm text-red-600 bg-red-50 p-2 rounded">
              {error}
            </div>
          )}
          
          <div className="flex justify-end gap-2">
            <Button
              type="button"
              variant="outline"
              onClick={() => setIsOpen(false)}
              disabled={isLoading}
            >
              Cancel
            </Button>
            <Button type="submit" disabled={isLoading}>
              {isLoading ? 'Creating Gift...' : 'Create Gift'}
            </Button>
          </div>
        </form>
      </DialogContent>
    </Dialog>
  )
}