import { Routes } from '@angular/router';
import { LoginComponent } from './pages/login/login.component';
import { MainComponent } from './pages/main/main.component';
import { WelcomeComponent } from './pages/welcome/welcome.component';
import { RbacComponent } from './pages/rbac/rbac.component';
import { RbacSimulatorComponent } from './pages/rbac/simulator.component';
import { AuditComponent } from './pages/audit/audit.component';
import { ActionsComponent } from './pages/actions/actions.component';
import { SessionComponent } from './pages/session/session.component';

export const routes: Routes = [
  { path: 'login', component: LoginComponent },
  {
    path: '',
    component: MainComponent,
    children: [
      { path: '', pathMatch: 'full', redirectTo: 'welcome' },
      {
        path: 'welcome',
        component: WelcomeComponent,
        data: { toolbar: { shadow: true, sticky: true } },
      },
      {
        path: 'rbac',
        component: RbacComponent,
        data: { toolbar: { shadow: false, sticky: false } },
      },
      {
        path: 'rbac/simulator',
        component: RbacSimulatorComponent,
        data: { toolbar: { shadow: true, sticky: true } },
      },
      {
        path: 'sessions',
        component: SessionComponent,
        data: { toolbar: { shadow: true, sticky: true } },
      },
      {
        path: 'audit',
        component: AuditComponent,
        data: { toolbar: { shadow: true, sticky: true } },
      },
      {
        path: 'network/dns',
        loadComponent: () => import('./pages/dns/dns.component').then((m) => m.DnsComponent),
        data: { toolbar: { shadow: false, sticky: false } },
      },
      {
        path: 'network/ip',
        loadComponent: () => import('./pages/ip/ip.component').then((m) => m.IpComponent),
        data: { toolbar: { shadow: false, sticky: false } },
      },
      {
        path: 'network/ip/sync',
        loadComponent: () => import('./pages/ip/sync.component').then((m) => m.IpSyncComponent),
        data: { toolbar: { shadow: true, sticky: true } },
      },
      {
        path: 'network/site',
        loadComponent: () => import('./pages/site/site.component').then((m) => m.SiteComponent),
        data: { toolbar: { shadow: false, sticky: false } },
      },
      {
        path: 'network/analysis',
        loadComponent: () =>
          import('./pages/network/analysis/analysis.component').then((m) => m.AnalysisComponent),
        data: { toolbar: { shadow: true, sticky: true } },
      },
      {
        path: 'network/intelligence',
        loadComponent: () =>
          import('./pages/network/intelligence/intelligence.component').then(
            (m) => m.IntelligenceComponent,
          ),
        data: { toolbar: { shadow: true, sticky: true } },
      },
      {
        path: 'actions',
        component: ActionsComponent,
        data: { toolbar: { shadow: false, sticky: false } },
      },
    ],
  },
];
