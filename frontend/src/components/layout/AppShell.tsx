import {
  AppBar,
  Box,
  Drawer,
  IconButton,
  List,
  ListItemButton,
  ListItemText,
  Toolbar,
  Typography,
  useMediaQuery,
} from '@mui/material';
import MenuIcon from '@mui/icons-material/Menu';
import { useMemo, useState } from 'react';
import { NavLink, Outlet } from 'react-router-dom';
import { tokens } from '../../styles/tokens';
import { theme } from '../../styles/theme';

const drawerWidth = 240;

const navItems = [
  { label: 'Dashboard', path: '/' },
  { label: 'Order Ticket', path: '/order-ticket' },
  { label: 'Blotter', path: '/blotter' },
  { label: 'Portfolio', path: '/portfolio' },
  { label: 'Settings', path: '/settings' },
];

export function AppShell() {
  const [mobileOpen, setMobileOpen] = useState(false);
  const isMobile = useMediaQuery(theme.breakpoints.down('md'));

  const drawerContent = useMemo(
    () => (
      <Box sx={{ padding: tokens.spacing.md }}>
        <Typography variant="overline" color="text.secondary">
          NAVIGATION
        </Typography>
        <List>
          {navItems.map((item) => (
            <NavLink key={item.path} to={item.path} style={{ textDecoration: 'none' }}>
              {({ isActive }) => (
                <ListItemButton
                  selected={isActive}
                  onClick={() => setMobileOpen(false)}
                  sx={{
                    borderRadius: tokens.radius.sm,
                    marginBottom: tokens.spacing.xs,
                    color: tokens.colors.textMuted,
                    border: '1px solid transparent',
                    '&.Mui-selected': {
                      backgroundColor: tokens.colors.surface,
                      color: tokens.colors.text,
                      border: `1px solid ${tokens.colors.border}`,
                    },
                  }}
                >
                  <ListItemText primary={item.label} />
                </ListItemButton>
              )}
            </NavLink>
          ))}
        </List>
      </Box>
    ),
    []
  );

  return (
    <Box sx={{ display: 'flex', minHeight: '100vh', backgroundColor: tokens.colors.bg }}>
      <AppBar
        position="fixed"
        color="transparent"
        elevation={0}
        sx={{ borderBottom: `1px solid ${tokens.colors.border}` }}
      >
        <Toolbar>
          {isMobile && (
            <IconButton color="inherit" edge="start" onClick={() => setMobileOpen(true)}>
              <MenuIcon />
            </IconButton>
          )}
          <Typography variant="h6" sx={{ fontWeight: tokens.typography.weight.semibold }}>
            Jax Trading UI
          </Typography>
          <Box sx={{ marginLeft: 'auto' }}>
            <Typography variant="body2" color="text.secondary">
              Live session - Connected
            </Typography>
          </Box>
        </Toolbar>
      </AppBar>

      <Box component="nav" sx={{ width: { md: drawerWidth }, flexShrink: { md: 0 } }}>
        <Drawer
          variant={isMobile ? 'temporary' : 'permanent'}
          open={isMobile ? mobileOpen : true}
          onClose={() => setMobileOpen(false)}
          ModalProps={{ keepMounted: true }}
          sx={{
            '& .MuiDrawer-paper': {
              width: drawerWidth,
              boxSizing: 'border-box',
              backgroundColor: tokens.colors.surface,
              borderRight: `1px solid ${tokens.colors.border}`,
            },
          }}
        >
          <Toolbar />
          {drawerContent}
        </Drawer>
      </Box>

      <Box component="main" sx={{ flexGrow: 1, padding: tokens.spacing.xl }}>
        <Toolbar />
        <Outlet />
      </Box>
    </Box>
  );
}
