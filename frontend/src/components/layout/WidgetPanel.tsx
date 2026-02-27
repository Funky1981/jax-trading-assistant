import { ReactNode, useLayoutEffect, useRef } from 'react';
import { GripVertical } from 'lucide-react';
import {
  Card,
  CardHeader,
  CardTitle,
  CardContent,
} from '@/components/ui/card';
import { cn } from '@/lib/utils';

interface WidgetPanelProps {
  /** Content to render inside the widget */
  children: ReactNode;
  /** Title displayed in the widget header */
  title: string;
  /** Unique identifier for the widget */
  id: string;
  /** Shows grip handle when true for drag functionality */
  isEditable?: boolean;
  /** Notify parent when content height changes (used to autosize widgets) */
  onHeightChange?: (id: string, height: number) => void;
  /** Additional className for the card */
  className?: string;
}

/**
 * Wrapper component for individual widgets within the WidgetGrid.
 * Includes a drag handle that's visible only in edit mode.
 */
export function WidgetPanel({
  children,
  title,
  id,
  isEditable = false,
  onHeightChange,
  className,
}: WidgetPanelProps) {
  const headerRef = useRef<HTMLDivElement | null>(null);
  const contentRef = useRef<HTMLDivElement | null>(null);
  const lastHeightRef = useRef<number>(0);

  useLayoutEffect(() => {
    if (!onHeightChange) {
      return;
    }

    const measure = () => {
      const headerHeight = headerRef.current?.offsetHeight ?? 0;
      const contentHeight = contentRef.current?.scrollHeight ?? 0;
      const nextHeight = Math.max(0, Math.ceil(headerHeight + contentHeight));
      if (Math.abs(nextHeight - lastHeightRef.current) < 4) {
        return;
      }
      lastHeightRef.current = nextHeight;
      onHeightChange(id, nextHeight);
    };

    measure();
    const observer = new ResizeObserver(measure);
    if (headerRef.current) {
      observer.observe(headerRef.current);
    }
    if (contentRef.current) {
      observer.observe(contentRef.current);
    }
    return () => observer.disconnect();
  }, [id, onHeightChange]);

  return (
    <Card
      id={id}
      className={cn(
        'widget-panel h-full w-full overflow-hidden',
        'flex flex-col',
        'bg-card border-border',
        'transition-shadow duration-200',
        isEditable && 'hover:shadow-lg hover:border-primary/50',
        className
      )}
    >
      <CardHeader ref={headerRef} className="flex-shrink-0 py-3 px-4">
        <div className="flex items-center gap-2">
          {isEditable && (
            <div
              className={cn(
                'widget-drag-handle',
                'cursor-grab active:cursor-grabbing',
                'flex items-center justify-center',
                'p-1 -ml-1 rounded',
                'text-muted-foreground hover:text-foreground',
                'hover:bg-muted/50',
                'transition-colors duration-150'
              )}
              title="Drag to reposition"
            >
              <GripVertical className="h-4 w-4" />
            </div>
          )}
          <CardTitle className="text-sm font-medium leading-none">
            {title}
          </CardTitle>
        </div>
      </CardHeader>
      <CardContent ref={contentRef} className="flex-1 min-h-0 p-4 pt-0 overflow-hidden">
        {children}
      </CardContent>
    </Card>
  );
}
