import { ElNotification } from 'element-plus-message'
import type { NotificationOptionsTyped } from 'element-plus-message'

type NotificationType = 'primary' | 'success' | 'info' | 'warning' | 'error'

export type MessageOptions = Partial<Omit<NotificationOptionsTyped, 'message'>> & {
  description?: NotificationOptionsTyped['message']
}

function notify(type: NotificationType, title: string, options: MessageOptions = {}) {
  const { description, ...notificationOptions } = options

  return ElNotification[type]({
    ...notificationOptions,
    title,
    ...(description === undefined ? {} : { message: description }),
  })
}

/** Compatibility wrapper for the app's existing toast calls. */
export const toast = {
  primary: (title: string, options?: MessageOptions) => notify('primary', title, options),
  success: (title: string, options?: MessageOptions) => notify('success', title, options),
  info: (title: string, options?: MessageOptions) => notify('info', title, options),
  warning: (title: string, options?: MessageOptions) => notify('warning', title, options),
  error: (title: string, options?: MessageOptions) => notify('error', title, options),
}
