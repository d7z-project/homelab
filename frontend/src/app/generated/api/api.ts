export * from './audit.service';
import { AuditService } from './audit.service';
export * from './auth.service';
import { AuthService } from './auth.service';
export * from './dns.service';
import { DnsService } from './dns.service';
export * from './rbac.service';
import { RbacService } from './rbac.service';
export const APIS = [AuditService, AuthService, DnsService, RbacService];
