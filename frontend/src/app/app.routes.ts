import { Routes } from '@angular/router';
import { LoginComponent } from './pages/login/login.component';
import { MainComponent } from './pages/main/main.component';
import { WelcomeComponent } from './pages/welcome/welcome.component';
import { RbacComponent } from './pages/rbac/rbac.component';
import { RbacSimulatorComponent } from './pages/rbac/simulator.component';
import { AuditComponent } from './pages/audit/audit.component';
import { DnsComponent } from './pages/dns/dns.component';
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
      { path: 'rbac/simulator', component: RbacSimulatorComponent },
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
      { path: 'dns', component: DnsComponent, data: { toolbar: { shadow: false, sticky: false } } },
      {
        path: 'ip',
        loadComponent: () => import('./pages/ip/ip.component').then((m) => m.IpComponent),
        data: { toolbar: { shadow: false, sticky: false } },
      },
      {
        path: 'actions',
        component: ActionsComponent,
        data: { toolbar: { shadow: false, sticky: false } },
      },
    ],
  },
];
