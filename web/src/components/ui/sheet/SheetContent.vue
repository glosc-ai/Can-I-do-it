<script setup lang="ts">
import type { DialogContentEmits, DialogContentProps } from 'reka-ui'
import type { HTMLAttributes } from 'vue'
import { reactiveOmit } from '@vueuse/core'
import { XIcon } from '@lucide/vue'
import { DialogClose, DialogContent, DialogPortal, useForwardPropsEmits } from 'reka-ui'
import { cn } from '@/lib/utils'
import SheetOverlay from './SheetOverlay.vue'

interface SheetContentProps extends DialogContentProps {
  class?: HTMLAttributes['class']
  side?: 'top' | 'right' | 'bottom' | 'left'
}

const props = withDefaults(defineProps<SheetContentProps>(), { side: 'right' })
const emits = defineEmits<DialogContentEmits>()

const delegatedProps = reactiveOmit(props, 'class', 'side')
const forwarded = useForwardPropsEmits(delegatedProps, emits)
</script>

<template>
  <DialogPortal>
    <SheetOverlay />
    <DialogContent
      data-slot="sheet-content"
      v-bind="forwarded"
      :class="cn(
        'fixed z-50 flex flex-col gap-4 bg-background shadow-lg transition ease-in-out',
        'data-[state=open]:animate-in data-[state=open]:duration-500 data-[state=closed]:animate-out data-[state=closed]:duration-300',
        {
          'inset-y-0 right-0 h-full w-3/4 border-l data-[state=open]:slide-in-from-right data-[state=closed]:slide-out-to-right sm:max-w-sm': side === 'right',
          'inset-y-0 left-0 h-full w-3/4 border-r data-[state=open]:slide-in-from-left data-[state=closed]:slide-out-to-left sm:max-w-sm': side === 'left',
          'inset-x-0 top-0 h-auto border-b data-[state=open]:slide-in-from-top data-[state=closed]:slide-out-to-top': side === 'top',
          'inset-x-0 bottom-0 h-auto border-t data-[state=open]:slide-in-from-bottom data-[state=closed]:slide-out-to-bottom': side === 'bottom',
        },
        props.class,
      )"
    >
      <slot />
      <DialogClose
        class="absolute top-4 right-4 rounded-xs opacity-70 ring-offset-background transition-opacity hover:opacity-100 focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2 disabled:pointer-events-none data-[state=open]:bg-secondary"
      >
        <XIcon class="size-4" />
        <span class="sr-only">关闭</span>
      </DialogClose>
    </DialogContent>
  </DialogPortal>
</template>
