# ATHENA Web Dashboard Implementation

## Overview

The ATHENA web dashboard is a comprehensive Next.js 16 application with TypeScript, providing a modern UI for managing Arduino template provisioning, device monitoring, and OTA firmware updates.

## Architecture

### Technology Stack
- **Framework**: Next.js 16.0.5 with React 19.2.0
- **Language**: TypeScript 5
- **Styling**: Tailwind CSS 4 with PostCSS
- **Testing**: Jest 29 with React Testing Library
- **Linting**: ESLint 9
- **Build**: SWC compiler with React Compiler optimization

### Project Structure

```
web-dashboard/
├── src/
│   ├── app/                    # Next.js app router pages
│   │   ├── layout.tsx          # Root layout with metadata
│   │   ├── page.tsx            # Home redirect to login
│   │   ├── login/              # Authentication page
│   │   ├── dashboard/          # Overview dashboard
│   │   ├── templates/          # Template catalog
│   │   │   └── [id]/           # Template detail page
│   │   ├── provisioning/       # Device provisioning
│   │   │   └── devices/        # Device management
│   │   │       └── [id]/       # Device detail page
│   │   ├── monitoring/         # Device monitoring
│   │   │   └── alerts/         # Alert management
│   │   └── ota/                # OTA firmware updates
│   │       └── deployments/    # Deployment management
│   │           └── [id]/       # Deployment detail
│   ├── components/             # Reusable React components
│   │   ├── layout/             # Layout components
│   │   ├── templates/          # Template-related components
│   │   ├── devices/            # Device-related components
│   │   ├── monitoring/         # Monitoring components
│   │   └── ota/                # OTA components
│   ├── hooks/                  # Custom React hooks
│   │   └── useAuth.ts          # Authentication hook
│   ├── lib/                    # Utility libraries
│   │   ├── auth/               # Authentication client
│   │   ├── templates/          # Template API client
│   │   ├── devices/            # Device API client
│   │   ├── telemetry/          # Telemetry API client
│   │   └── ota/                # OTA API client
│   └── app/globals.css         # Global styles
├── public/                     # Static assets
├── jest.config.js              # Jest configuration
├── jest.setup.js               # Jest setup file
├── next.config.ts              # Next.js configuration
├── tsconfig.json               # TypeScript configuration
└── package.json                # Dependencies

```

## Features Implemented

### 10.1 React Foundation with Authentication ✅
- **Next.js Setup**: Modern app router with TypeScript
- **Authentication**: JWT-based login with token storage
- **Authorization**: Protected routes with `useAuth` hook
- **Responsive Layout**: Mobile-first design with Tailwind CSS
- **Navigation**: Sidebar navigation with active state tracking

**Key Files**:
- `src/app/login/page.tsx` - Login page with auto-login
- `src/lib/auth/client.ts` - Authentication API client
- `src/hooks/useAuth.ts` - Authentication state hook
- `src/components/layout/dashboard-shell.tsx` - Main layout wrapper

### 10.2 Template Management Interface ✅
- **Template Catalog**: Grid view with pagination
- **Filtering**: Search, category, and tag-based filtering
- **Template Details**: Full template information with metadata
- **Configuration Forms**: JSON Schema-based dynamic form generation
- **Preview**: Wiring diagrams, documentation, and code examples
- **Template Cards**: Quick view with version and author info

**Key Files**:
- `src/app/templates/page.tsx` - Template catalog
- `src/app/templates/[id]/page.tsx` - Template detail page
- `src/components/templates/template-card.tsx` - Template card component
- `src/components/templates/template-config-form.tsx` - Dynamic form builder
- `src/components/templates/template-preview.tsx` - Preview with tabs
- `src/components/templates/template-filters.tsx` - Filter controls
- `src/lib/templates/client.ts` - Template API client

### 10.3 Device Provisioning Interface ✅
- **Device List**: Paginated device catalog with status indicators
- **Device Details**: Comprehensive device information and configuration
- **Provisioning Workflow**: Multi-step compilation and flashing process
- **Serial Monitor**: Real-time serial communication with auto-scroll
- **Device Registration**: Create and configure new devices
- **Status Tracking**: Real-time job progress monitoring

**Key Files**:
- `src/app/provisioning/page.tsx` - Device list
- `src/app/provisioning/devices/[id]/page.tsx` - Device detail
- `src/components/devices/device-card.tsx` - Device card
- `src/components/devices/provisioning-workflow.tsx` - Provisioning workflow
- `src/components/devices/serial-monitor.tsx` - Serial communication
- `src/components/devices/device-filters.tsx` - Filter controls
- `src/lib/devices/client.ts` - Device API client

### 10.4 Device Monitoring Dashboard ✅
- **Device Health Grid**: Real-time device status overview
- **Telemetry Visualization**: SVG-based line charts for metrics
- **Alert Management**: Alert list with severity and status filtering
- **Alert Rules**: Create and manage alert rules
- **Real-time Updates**: 10-second polling for live data
- **Alert Actions**: Acknowledge, resolve, and manage alerts

**Key Files**:
- `src/app/monitoring/page.tsx` - Monitoring dashboard
- `src/app/monitoring/alerts/page.tsx` - Alert management
- `src/components/monitoring/device-health-grid.tsx` - Health overview
- `src/components/monitoring/telemetry-chart.tsx` - Chart visualization
- `src/components/monitoring/alert-list.tsx` - Alert list
- `src/lib/telemetry/client.ts` - Telemetry API client

### 10.5 OTA Management Interface ✅
- **Firmware Releases**: Upload and manage firmware versions
- **Release Management**: Version control with checksums and metadata
- **Deployment Configuration**: Create phased rollout strategies
- **Deployment Monitoring**: Real-time progress tracking
- **Rollback Interface**: Pause, resume, and rollback deployments
- **Target Groups**: Configure device groups for deployments

**Key Files**:
- `src/app/ota/page.tsx` - OTA dashboard
- `src/app/ota/deployments/page.tsx` - Deployment list
- `src/app/ota/deployments/[id]/page.tsx` - Deployment detail
- `src/components/ota/firmware-release-card.tsx` - Release card
- `src/components/ota/deployment-rollout.tsx` - Deployment controls
- `src/lib/ota/client.ts` - OTA API client

### 10.6 Frontend Tests and Validation ✅
- **Component Tests**: Jest + React Testing Library
- **User Interaction Tests**: Form submission, button clicks
- **Accessibility Tests**: axe-core integration
- **Responsive Design Tests**: Mobile and desktop layouts
- **API Integration Tests**: Mock API responses

**Test Coverage**:
- `src/components/devices/__tests__/device-card.test.tsx` - Device card tests
- `src/components/devices/__tests__/provisioning-workflow.test.tsx` - Workflow tests
- `src/components/templates/__tests__/template-card.test.tsx` - Template card tests
- `src/components/templates/__tests__/template-config-form.test.tsx` - Form tests
- `src/components/monitoring/__tests__/alert-list.test.tsx` - Alert tests
- `src/components/monitoring/__tests__/telemetry-chart.test.tsx` - Chart tests
- `src/components/ota/__tests__/firmware-release-card.test.tsx` - Release tests
- `src/components/ota/__tests__/deployment-rollout.test.tsx` - Deployment tests
- `src/components/layout/__tests__/dashboard-shell.test.tsx` - Layout tests

## API Integration

The dashboard communicates with the ATHENA backend via REST APIs:

### Base URL
- Development: `http://localhost:8080`
- Production: Configurable via `NEXT_PUBLIC_API_BASE_URL`

### API Endpoints

#### Authentication
- `POST /api/v1/auth/login` - User login
- `POST /api/v1/auth/refresh` - Token refresh
- `POST /api/v1/auth/logout` - User logout
- `GET /api/v1/auth/me` - Current user info

#### Templates
- `GET /api/v1/templates` - List templates with pagination
- `GET /api/v1/templates/{id}` - Get template details
- `POST /api/v1/templates` - Create template
- `PUT /api/v1/templates/{id}` - Update template

#### Devices
- `GET /api/v1/devices` - List devices with pagination
- `GET /api/v1/devices/{id}` - Get device details
- `POST /api/v1/devices` - Create device
- `PUT /api/v1/devices/{id}` - Update device
- `POST /api/v1/devices/provision` - Start provisioning
- `GET /api/v1/devices/{id}/provisioning-job/{jobId}` - Get job status
- `GET /api/v1/devices/{id}/serial` - Get serial messages
- `POST /api/v1/devices/{id}/serial` - Send serial message

#### Telemetry
- `GET /api/v1/telemetry/devices/{id}/health` - Device health
- `GET /api/v1/telemetry/health` - All devices health
- `GET /api/v1/telemetry/alerts` - List alerts
- `POST /api/v1/telemetry/alerts/{id}/acknowledge` - Acknowledge alert
- `POST /api/v1/telemetry/alerts/{id}/resolve` - Resolve alert
- `GET /api/v1/telemetry/alert-rules` - List alert rules
- `POST /api/v1/telemetry/alert-rules` - Create alert rule
- `PUT /api/v1/telemetry/alert-rules/{id}` - Update alert rule
- `DELETE /api/v1/telemetry/alert-rules/{id}` - Delete alert rule

#### OTA
- `GET /api/v1/ota/releases` - List firmware releases
- `POST /api/v1/ota/releases` - Upload firmware release
- `GET /api/v1/ota/deployments` - List deployments
- `POST /api/v1/ota/deployments` - Create deployment
- `GET /api/v1/ota/deployments/{id}` - Get deployment details
- `POST /api/v1/ota/deployments/{id}/pause` - Pause deployment
- `POST /api/v1/ota/deployments/{id}/resume` - Resume deployment
- `POST /api/v1/ota/deployments/{id}/rollback` - Rollback deployment

## Running the Dashboard

### Development
```bash
cd web-dashboard
npm install
npm run dev
```
The dashboard will be available at `http://localhost:3000`

### Production Build
```bash
npm run build
npm start
```

### Testing
```bash
# Run all tests
npm test

# Run tests in watch mode
npm run test:watch

# Generate coverage report
npm run test:coverage
```

### Linting
```bash
npm run lint
```

## Environment Variables

Create a `.env.local` file:

```env
# API Configuration
NEXT_PUBLIC_API_BASE_URL=http://localhost:8080

# Optional: Analytics, error tracking, etc.
```

## Design System

### Colors
- **Primary**: Zinc (gray) - `#09090b` to `#fafafa`
- **Success**: Green - `#10b981`
- **Warning**: Yellow - `#f59e0b`
- **Error**: Red - `#ef4444`
- **Info**: Blue - `#3b82f6`

### Typography
- **Sans**: Geist (Google Fonts)
- **Mono**: Geist Mono (Google Fonts)

### Components
- **Buttons**: Primary, secondary, danger variants
- **Forms**: Text input, select, checkbox, textarea
- **Cards**: Rounded borders with subtle shadows
- **Badges**: Status indicators with color coding
- **Modals**: Overlay dialogs for confirmations

## Performance Optimizations

1. **React Compiler**: Enabled via Babel plugin
2. **Image Optimization**: Next.js Image component
3. **Code Splitting**: Automatic per-route
4. **CSS Optimization**: Tailwind CSS purging
5. **Bundle Analysis**: Use `next/bundle-analyzer`

## Accessibility

- **WCAG 2.1 AA Compliance**: Axe-core integration
- **Semantic HTML**: Proper heading hierarchy
- **ARIA Labels**: Form labels and descriptions
- **Keyboard Navigation**: Full keyboard support
- **Color Contrast**: WCAG AA compliant

## Browser Support

- Chrome/Edge 90+
- Firefox 88+
- Safari 14+
- Mobile browsers (iOS Safari 14+, Chrome Android)

## Deployment

The dashboard can be deployed to:
- **Vercel**: Recommended (native Next.js support)
- **Netlify**: Via Next.js adapter
- **Docker**: Using `Dockerfile.dashboard`
- **Traditional Servers**: Node.js 18+

## Future Enhancements

1. **Dark Mode**: Theme toggle
2. **Internationalization**: Multi-language support
3. **Advanced Analytics**: Device metrics dashboard
4. **Real-time WebSocket**: Live updates instead of polling
5. **Mobile App**: React Native version
6. **Offline Support**: Service Worker integration
7. **Advanced Filtering**: Complex query builder
8. **Custom Dashboards**: User-configurable widgets

## Troubleshooting

### Port Already in Use
```bash
# Use a different port
npm run dev -- -p 3001
```

### API Connection Issues
- Check `NEXT_PUBLIC_API_BASE_URL` environment variable
- Verify backend is running on the correct port
- Check CORS headers from backend

### Build Errors
```bash
# Clear Next.js cache
rm -rf .next
npm run build
```

### Test Failures
```bash
# Clear Jest cache
npm test -- --clearCache
```

## Contributing

1. Follow TypeScript strict mode
2. Write tests for new components
3. Use Tailwind CSS for styling
4. Follow the existing component structure
5. Update this documentation for new features

## License

Part of the ATHENA project - Arduino Template Hub with Natural-Language Provisioning
