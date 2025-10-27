'use client'

import { useState, useEffect } from 'react'
import { useAuth } from '@/lib/auth'
import { getUserSubscriptions, deleteSubscription, updateSubscription } from '@/lib/api'
import { Button } from '@/components/ui/Button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { 
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { 
  Dialog, 
  DialogContent, 
  DialogDescription, 
  DialogHeader, 
  DialogTitle, 
  DialogTrigger 
} from '@/components/ui/dialog'
import { Label } from '@/components/ui/label'
import { Checkbox } from '@/components/ui/checkbox'
import { Bell, Settings, Trash2, Edit } from 'lucide-react'
import Link from 'next/link'

interface Subscription {
  id: string
  type: string
  target_id: string
  target_name: string
  events: string[]
  frequency: string
  filter_tags?: string[]
  filter_rating?: string[]
  is_active: boolean
  created_at: string
  updated_at: string
}

export default function SubscriptionsPage() {
  const { user, token } = useAuth()
  const [subscriptions, setSubscriptions] = useState<Subscription[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [editingSubscription, setEditingSubscription] = useState<Subscription | null>(null)
  const [isEditDialogOpen, setIsEditDialogOpen] = useState(false)

  // Edit form state
  const [editFrequency, setEditFrequency] = useState<'immediate' | 'daily' | 'weekly'>('immediate')
  const [editEvents, setEditEvents] = useState<string[]>([])
  const [isUpdating, setIsUpdating] = useState(false)

  useEffect(() => {
    if (user && token) {
      loadSubscriptions()
    } else {
      setLoading(false)
    }
  }, [user, token])

  const loadSubscriptions = async () => {
    try {
      setLoading(true)
      const data = await getUserSubscriptions(token!)
      setSubscriptions(data.subscriptions || [])
    } catch (error) {
      setError(error instanceof Error ? error.message : 'Failed to load subscriptions')
    } finally {
      setLoading(false)
    }
  }

  const handleDelete = async (subscriptionId: string) => {
    if (!confirm('Are you sure you want to delete this subscription?')) {
      return
    }

    try {
      await deleteSubscription(subscriptionId, token!)
      setSubscriptions(prev => prev.filter(sub => sub.id !== subscriptionId))
    } catch (error) {
      setError(error instanceof Error ? error.message : 'Failed to delete subscription')
    }
  }

  const handleEdit = (subscription: Subscription) => {
    setEditingSubscription(subscription)
    setEditFrequency(subscription.frequency as 'immediate' | 'daily' | 'weekly')
    setEditEvents(subscription.events)
    setIsEditDialogOpen(true)
  }

  const handleUpdate = async () => {
    if (!editingSubscription) return

    try {
      setIsUpdating(true)
      await updateSubscription(editingSubscription.id, {
        frequency: editFrequency,
        events: editEvents
      }, token!)
      
      // Update local state
      setSubscriptions(prev => prev.map(sub => 
        sub.id === editingSubscription.id 
          ? { ...sub, frequency: editFrequency, events: editEvents }
          : sub
      ))
      
      setIsEditDialogOpen(false)
      setEditingSubscription(null)
    } catch (error) {
      setError(error instanceof Error ? error.message : 'Failed to update subscription')
    } finally {
      setIsUpdating(false)
    }
  }

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

  const handleEventToggle = (eventId: string, checked: boolean) => {
    if (checked) {
      setEditEvents(prev => [...prev, eventId])
    } else {
      setEditEvents(prev => prev.filter(id => id !== eventId))
    }
  }

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleDateString('en-US', {
      year: 'numeric',
      month: 'short',
      day: 'numeric'
    })
  }

  const getTypeIcon = (type: string) => {
    switch (type) {
      case 'work':
        return '📖'
      case 'author':
        return '👤'
      case 'series':
        return '📚'
      case 'tag':
        return '🏷️'
      case 'collection':
        return '📁'
      default:
        return '🔔'
    }
  }

  if (!user) {
    return (
      <div className="container mx-auto px-4 py-8">
        <div className="text-center">
          <h1 className="text-2xl font-bold mb-4">Subscriptions</h1>
          <p className="text-muted-foreground mb-4">You need to be logged in to view your subscriptions.</p>
          <Link href="/login">
            <Button>Log In</Button>
          </Link>
        </div>
      </div>
    )
  }

  if (loading) {
    return (
      <div className="container mx-auto px-4 py-8">
        <div className="text-center">
          <Bell className="h-8 w-8 animate-pulse mx-auto mb-4" />
          <p>Loading your subscriptions...</p>
        </div>
      </div>
    )
  }

  return (
    <div className="container mx-auto px-4 py-8">
      <div className="mb-8">
        <h1 className="text-3xl font-bold mb-2">Your Subscriptions</h1>
        <p className="text-muted-foreground">
          Manage your notifications for works, authors, and more.
        </p>
      </div>

      {error && (
        <div className="bg-red-50 border border-red-200 text-red-600 px-4 py-3 rounded mb-6">
          {error}
        </div>
      )}

      {subscriptions.length === 0 ? (
        <Card>
          <CardContent className="text-center py-8">
            <Bell className="h-12 w-12 text-muted-foreground mx-auto mb-4" />
            <h3 className="text-lg font-semibold mb-2">No subscriptions yet</h3>
            <p className="text-muted-foreground mb-4">
              Start following your favorite works and authors to get notified of updates.
            </p>
            <Link href="/works">
              <Button>Browse Works</Button>
            </Link>
          </CardContent>
        </Card>
      ) : (
        <div className="grid gap-4">
          {subscriptions.map((subscription) => (
            <Card key={subscription.id} className={!subscription.is_active ? 'opacity-60' : ''}>
              <CardHeader>
                <div className="flex items-start justify-between">
                  <div className="flex items-start space-x-3">
                    <span className="text-2xl">{getTypeIcon(subscription.type)}</span>
                    <div>
                      <CardTitle className="text-lg">
                        {subscription.target_name}
                      </CardTitle>
                      <CardDescription>
                        {subscription.type.charAt(0).toUpperCase() + subscription.type.slice(1)} subscription
                        {' • '}
                        Created {formatDate(subscription.created_at)}
                      </CardDescription>
                    </div>
                  </div>
                  <div className="flex items-center space-x-2">
                    <Badge variant={subscription.frequency === 'immediate' ? 'default' : 'secondary'}>
                      {subscription.frequency}
                    </Badge>
                    {!subscription.is_active && (
                      <Badge variant="outline">Paused</Badge>
                    )}
                  </div>
                </div>
              </CardHeader>
              <CardContent>
                <div className="space-y-4">
                  <div>
                    <Label className="text-sm font-medium">Events:</Label>
                    <div className="flex flex-wrap gap-1 mt-1">
                      {subscription.events.map((event) => (
                        <Badge key={event} variant="outline" className="text-xs">
                          {event.replace('_', ' ').replace(/\b\w/g, l => l.toUpperCase())}
                        </Badge>
                      ))}
                    </div>
                  </div>

                  <div className="flex items-center space-x-2">
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => handleEdit(subscription)}
                    >
                      <Edit className="h-4 w-4 mr-1" />
                      Edit
                    </Button>
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => handleDelete(subscription.id)}
                      className="text-red-600 hover:text-red-700 hover:bg-red-50"
                    >
                      <Trash2 className="h-4 w-4 mr-1" />
                      Delete
                    </Button>
                  </div>
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      )}

      {/* Edit Subscription Dialog */}
      <Dialog open={isEditDialogOpen} onOpenChange={setIsEditDialogOpen}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>Edit Subscription</DialogTitle>
            <DialogDescription>
              Update your notification preferences for {editingSubscription?.target_name}.
            </DialogDescription>
          </DialogHeader>
          
          {editingSubscription && (
            <div className="space-y-4">
              <div className="space-y-2">
                <Label>Notification Frequency</Label>
                <Select value={editFrequency} onValueChange={(value: 'immediate' | 'daily' | 'weekly') => setEditFrequency(value)}>
                  <SelectTrigger>
                    <SelectValue />
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
                  {getAvailableEvents(editingSubscription.type).map((event) => (
                    <div key={event.id} className="flex items-start space-x-2">
                      <Checkbox
                        id={event.id}
                        checked={editEvents.includes(event.id)}
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
              
              <div className="flex justify-end gap-2">
                <Button
                  type="button"
                  variant="outline"
                  onClick={() => setIsEditDialogOpen(false)}
                  disabled={isUpdating}
                >
                  Cancel
                </Button>
                <Button 
                  onClick={handleUpdate} 
                  disabled={isUpdating || editEvents.length === 0}
                >
                  {isUpdating ? 'Updating...' : 'Update'}
                </Button>
              </div>
            </div>
          )}
        </DialogContent>
      </Dialog>
    </div>
  )
}