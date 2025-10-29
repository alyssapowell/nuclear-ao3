'use client'

import { useState, useEffect } from 'react'
import { useAuth } from '@/lib/auth'
import { 
  createSubscription, 
  deleteSubscription, 
  checkSubscriptionStatus,
  CreateSubscriptionRequest 
} from '@/lib/api'
import { Button } from '@/components/ui/Button'
import { 
  Dialog, 
  DialogContent, 
  DialogDescription, 
  DialogHeader, 
  DialogTitle, 
  DialogTrigger 
} from '@/components/ui/dialog'
import { Label } from '@/components/ui/label'
import { 
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Checkbox } from '@/components/ui/checkbox'
import { Bell, BellOff } from 'lucide-react'

interface SubscriptionButtonProps {
  type: 'work' | 'author' | 'series' | 'tag' | 'collection';
  targetId: string;
  targetName: string;
  className?: string;
}

export function SubscriptionButton({ type, targetId, targetName, className }: SubscriptionButtonProps) {
  const { user, token } = useAuth()
  const [isSubscribed, setIsSubscribed] = useState(false)
  const [subscriptionId, setSubscriptionId] = useState<string | null>(null)
  const [isOpen, setIsOpen] = useState(false)
  const [isLoading, setIsLoading] = useState(false)
  const [isCheckingStatus, setIsCheckingStatus] = useState(true)
  const [error, setError] = useState<string | null>(null)

  // Form state
  const [frequency, setFrequency] = useState<'immediate' | 'daily' | 'weekly'>('immediate')
  const [selectedEvents, setSelectedEvents] = useState<string[]>([])

  // Available events based on subscription type
  const getAvailableEvents = (subscriptionType: string) => {
    switch (subscriptionType) {
      case 'work':
        return [
          { id: 'work_updated', label: 'Work Updated', description: 'When new chapters are posted' },
          { id: 'work_completed', label: 'Work Completed', description: 'When the work is marked complete' }
        ]
      case 'author':
        return [
          { id: 'new_work', label: 'New Work', description: 'When the author posts a new work' },
          { id: 'work_updated', label: 'Work Updated', description: 'When the author updates any work' }
        ]
      case 'series':
        return [
          { id: 'series_updated', label: 'Series Updated', description: 'When new works are added to the series' },
          { id: 'work_updated', label: 'Work Updated', description: 'When any work in the series is updated' }
        ]
      default:
        return [
          { id: 'new_work', label: 'New Work', description: 'When new works are posted' }
        ]
    }
  }

  const availableEvents = getAvailableEvents(type)

  useEffect(() => {
    if (user && token) {
      checkStatus()
    }
  }, [user, token, targetId, type])

  useEffect(() => {
    // Set default events when type changes
    const defaultEvents = availableEvents.map(event => event.id)
    setSelectedEvents(defaultEvents)
  }, [type])

  const checkStatus = async () => {
    try {
      setIsCheckingStatus(true)
      const result = await checkSubscriptionStatus(type, targetId, token || undefined)
      setIsSubscribed(result.subscribed)
      setSubscriptionId(result.subscription_id)
    } catch (error) {
      console.error('Failed to check subscription status:', error)
    } finally {
      setIsCheckingStatus(false)
    }
  }

  const handleSubscribe = async () => {
    if (!user || !token) return

    setIsLoading(true)
    setError(null)

    try {
      const subscriptionData: CreateSubscriptionRequest = {
        type,
        target_id: targetId,
        target_name: targetName,
        events: selectedEvents,
        frequency
      }

      const result = await createSubscription(subscriptionData, token || undefined)
      setIsSubscribed(true)
      setSubscriptionId(result.subscription.id)
      setIsOpen(false)
    } catch (error) {
      setError(error instanceof Error ? error.message : 'Failed to create subscription')
    } finally {
      setIsLoading(false)
    }
  }

  const handleUnsubscribe = async () => {
    if (!subscriptionId || !token) return

    setIsLoading(true)
    setError(null)

    try {
      await deleteSubscription(subscriptionId, token || undefined)
      setIsSubscribed(false)
      setSubscriptionId(null)
    } catch (error) {
      setError(error instanceof Error ? error.message : 'Failed to unsubscribe')
    } finally {
      setIsLoading(false)
    }
  }

  const handleEventToggle = (eventId: string, checked: boolean) => {
    if (checked) {
      setSelectedEvents(prev => [...prev, eventId])
    } else {
      setSelectedEvents(prev => prev.filter(id => id !== eventId))
    }
  }

  if (!user) {
    return null // Don't show for non-authenticated users
  }

  if (isCheckingStatus) {
    return (
      <Button variant="outline" size="sm" disabled className={className}>
        <Bell className="h-4 w-4 animate-pulse" />
        Checking...
      </Button>
    )
  }

  if (isSubscribed) {
    return (
      <Button 
        variant="outline" 
        size="sm" 
        onClick={handleUnsubscribe}
        disabled={isLoading}
        className={className}
      >
        <BellOff className="h-4 w-4" />
        {isLoading ? 'Unsubscribing...' : 'Unsubscribe'}
      </Button>
    )
  }

  return (
    <Dialog open={isOpen} onOpenChange={setIsOpen}>
      <DialogTrigger asChild>
        <Button variant="outline" size="sm" className={className}>
          <Bell className="h-4 w-4" />
          Subscribe
        </Button>
      </DialogTrigger>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Subscribe to {targetName}</DialogTitle>
          <DialogDescription>
            Get notified when there are updates to this {type}.
          </DialogDescription>
        </DialogHeader>
        
        <div className="space-y-4">
          <div className="space-y-2">
            <Label>Notification Frequency</Label>
            <Select value={frequency} onValueChange={(value) => setFrequency(value as 'immediate' | 'daily' | 'weekly')}>
              <SelectTrigger>
                <SelectValue placeholder="Select frequency" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="immediate">Immediate</SelectItem>
                <SelectItem value="daily">Daily Digest</SelectItem>
                <SelectItem value="weekly">Weekly Digest</SelectItem>
              </SelectContent>
            </Select>
          </div>

          <div className="space-y-2">
            <Label>Events to Subscribe To</Label>
            <div className="space-y-2">
              {availableEvents.map((event) => (
                <div key={event.id} className="flex items-start space-x-2">
                  <Checkbox
                    id={event.id}
                    checked={selectedEvents.includes(event.id)}
                    onCheckedChange={(checked) => handleEventToggle(event.id, checked as boolean)}
                  />
                  <div className="grid gap-1.5 leading-none">
                    <Label 
                      htmlFor={event.id}
                      className="text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70"
                    >
                      {event.label}
                    </Label>
                    <p className="text-xs text-muted-foreground">
                      {event.description}
                    </p>
                  </div>
                </div>
              ))}
            </div>
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
            <Button 
              onClick={handleSubscribe} 
              disabled={isLoading || selectedEvents.length === 0}
            >
              {isLoading ? 'Subscribing...' : 'Subscribe'}
            </Button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  )
}