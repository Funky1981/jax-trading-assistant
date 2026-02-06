import { ReactNode } from 'react';
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
  className,
}: WidgetPanelProps) {
  return (
    <Card
      id={id}
      className={cn(
        'h-full w-full overflow-hidden',
        'flex flex-col',
        'bg-card border-border',
        'transition-shadow duration-200',
        isEditable && 'hover:shadow-lg hover:border-primary/50',
        className
      )}
    >
      <CardHeader className="flex-shrink-0 py-3 px-4">
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
      <CardContent className="flex-1 min-h-0 overflow-auto p-4 pt-0">
        {children}
      </CardContent>
    </Card>
  );
}
