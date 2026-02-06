import { ReactNode, useMemo } from 'react';
import {
  ResponsiveGridLayout,
  useContainerWidth,
  type Layout,
  type ResponsiveLayouts,
} from 'react-grid-layout';
import 'react-grid-layout/css/styles.css';
import 'react-resizable/css/styles.css';
import '@/styles/widget-grid.css';
import { cn } from '@/lib/utils';

export type { Layout };
export type Layouts = ResponsiveLayouts<string>;

interface WidgetGridProps {
  /** Widget panels to render within the grid */
  children: ReactNode;
  /** Layout configuration for different breakpoints */
  layouts: Layouts;
  /** Callback when layout changes */
  onLayoutChange: (layouts: Layouts) => void;
  /** When true, allow drag/resize; when false, static */
  isEditable?: boolean;
  /** Additional className for the container */
  className?: string;
}

/** Breakpoints for responsive layout */
const BREAKPOINTS = { lg: 1200, md: 996, sm: 768, xs: 480 };

/** Number of columns at each breakpoint */
const COLS = { lg: 12, md: 10, sm: 6, xs: 4 };

/** Height of each row in pixels */
const ROW_HEIGHT = 60;

/** Container padding [horizontal, vertical] */
const CONTAINER_PADDING: [number, number] = [16, 16];

/** Margin between grid items [horizontal, vertical] */
const MARGIN: [number, number] = [16, 16];

/**
 * A customizable draggable widget grid component using react-grid-layout.
 * Supports responsive layouts with drag-and-drop and resize functionality.
 */
export function WidgetGrid({
  children,
  layouts,
  onLayoutChange,
  isEditable = false,
  className,
}: WidgetGridProps) {
  const { width, containerRef, mounted } = useContainerWidth();

  // Memoize drag/resize config to prevent unnecessary re-renders
  const dragConfig = useMemo(
    () => ({
      enabled: isEditable,
      handle: isEditable ? '.widget-drag-handle' : undefined,
    }),
    [isEditable]
  );

  const resizeConfig = useMemo(
    () => ({
      enabled: isEditable,
    }),
    [isEditable]
  );

  return (
    <div
      ref={containerRef}
      className={cn(
        'widget-grid-container',
        'w-full min-h-0',
        'bg-background',
        className
      )}
    >
      {mounted && (
        <ResponsiveGridLayout
          className="widget-grid"
          width={width}
          layouts={layouts}
          breakpoints={BREAKPOINTS}
          cols={COLS}
          rowHeight={ROW_HEIGHT}
          containerPadding={CONTAINER_PADDING}
          margin={MARGIN}
          onLayoutChange={(currentLayout: Layout, allLayouts: Layouts) => {
            onLayoutChange(allLayouts);
          }}
          dragConfig={dragConfig}
          resizeConfig={resizeConfig}
        >
          {children}
        </ResponsiveGridLayout>
      )}
    </div>
  );
}

export { BREAKPOINTS, COLS, ROW_HEIGHT };
