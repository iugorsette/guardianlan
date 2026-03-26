import { CommonModule, DatePipe, DecimalPipe } from '@angular/common';
import {
  ChangeDetectionStrategy,
  Component,
  DestroyRef,
  computed,
  inject,
  signal
} from '@angular/core';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { FormsModule } from '@angular/forms';
import { interval, startWith } from 'rxjs';

import { DashboardApiService } from './dashboard-api.service';
import { Alert, Device, DNSPolicy, Profile } from './models';
import { DeviceEvidence, DeviceInsight } from './models';
import { DASHBOARD_AUTO_REFRESH_MS } from './dashboard.tokens';

@Component({
  selector: 'app-root',
  imports: [CommonModule, FormsModule, DatePipe, DecimalPipe],
  templateUrl: './app.html',
  styleUrl: './app.css',
  changeDetection: ChangeDetectionStrategy.OnPush
})
export class App {
  protected readonly title = 'Guardian LAN';
  protected readonly api = inject(DashboardApiService);
  private readonly destroyRef = inject(DestroyRef);
  private readonly autoRefreshMs = inject(DASHBOARD_AUTO_REFRESH_MS);
  protected readonly profileDrafts = signal<Record<string, string>>({});
  protected readonly nameDrafts = signal<Record<string, string>>({});
  protected readonly dnsPolicyDrafts = signal<Record<string, DNSPolicyFormValue>>({});
  protected readonly selectedDeviceId = signal<string | null>(null);
  protected readonly summary = this.api.summary;
  protected readonly profiles = this.api.profiles;
  protected readonly devices = computed(() =>
    [...this.api.devices()].sort((left, right) => right.risk_score - left.risk_score)
  );
  protected readonly alerts = computed(() =>
    [...this.api.alerts()].sort(
      (left, right) =>
        new Date(right.created_at).getTime() - new Date(left.created_at).getTime()
    )
  );
  protected readonly dnsEvents = computed(() => this.api.dnsEvents().slice(0, 8));
  protected readonly flowEvents = computed(() => this.api.flowEvents().slice(0, 8));
  protected readonly cameraDevices = computed(() =>
    this.devices().filter((device) => device.device_type === 'camera')
  );
  protected readonly selectedDevice = computed(() => {
    const selectedDeviceId = this.selectedDeviceId();
    if (!selectedDeviceId) {
      return null;
    }

    return this.devices().find((device) => device.id === selectedDeviceId) ?? null;
  });

  constructor() {
    if (this.autoRefreshMs < 0) {
      return;
    }

    interval(this.autoRefreshMs)
      .pipe(startWith(0), takeUntilDestroyed(this.destroyRef))
      .subscribe(() => this.api.refresh());
  }

  protected trackById = (_: number, item: { id: string }) => item.id;

  protected profileFor(device: Device): string {
    return this.profileDrafts()[device.id] ?? device.profile_id;
  }

  protected nameFor(device: Device): string {
    return this.nameDrafts()[device.id] ?? device.display_name ?? '';
  }

  protected setProfile(deviceId: string, profileId: string): void {
    this.profileDrafts.update((drafts) => ({ ...drafts, [deviceId]: profileId }));
  }

  protected setDeviceName(deviceId: string, displayName: string): void {
    this.nameDrafts.update((drafts) => ({ ...drafts, [deviceId]: displayName }));
  }

  protected dnsPolicyFor(device: Device): DNSPolicyFormValue {
    return (
      this.dnsPolicyDrafts()[device.id] ?? {
        blockedDomains: (device.dns_policy_override?.blocked_domains ?? []).join('\n'),
        allowedDomains: (device.dns_policy_override?.allowed_domains ?? []).join('\n'),
        blockedCategories: (device.dns_policy_override?.blocked_categories ?? []).join('\n'),
        safeSearch: device.dns_policy_override?.safe_search ?? false
      }
    );
  }

  protected setDNSPolicyDraft(deviceId: string, patch: Partial<DNSPolicyFormValue>): void {
    const current = this.dnsPolicyDrafts()[deviceId] ?? {
      blockedDomains: '',
      allowedDomains: '',
      blockedCategories: '',
      safeSearch: false
    };
    this.dnsPolicyDrafts.update((drafts) => ({
      ...drafts,
      [deviceId]: { ...current, ...patch }
    }));
  }

  protected saveProfile(device: Device): void {
    const profileId = this.profileFor(device);
    if (profileId === device.profile_id) {
      return;
    }

    this.api.updateProfile(device.id, profileId);
  }

  protected saveDeviceName(device: Device): void {
    const displayName = this.nameFor(device).trim();
    if (displayName === (device.display_name ?? '')) {
      return;
    }

    this.api.updateDeviceName(device.id, displayName);
  }

  protected saveDNSPolicy(device: Device): void {
    const draft = this.dnsPolicyFor(device);
    this.api.updateDeviceDNSPolicy(device.id, {
      safe_search: draft.safeSearch,
      blocked_domains: this.splitPolicyLines(draft.blockedDomains),
      allowed_domains: this.splitPolicyLines(draft.allowedDomains),
      blocked_categories: this.splitPolicyLines(draft.blockedCategories)
    });
  }

  protected acknowledge(alert: Alert): void {
    if (alert.status !== 'open') {
      return;
    }

    this.api.acknowledgeAlert(alert.id);
  }

  protected latestInsight(deviceId: string): DeviceInsight | null {
    return this.api.deviceInsights()[deviceId]?.[0] ?? null;
  }

  protected evidenceFor(deviceId: string): DeviceEvidence | null {
    const insight = this.latestInsight(deviceId);
    if (!insight) {
      return null;
    }

    return insight.evidence as DeviceEvidence;
  }

  protected profileLabel(profileId: string): string {
    const profile = this.profiles().find((item) => item.id === profileId);
    return profile?.name ?? profileId;
  }

  protected profileOptions(): Profile[] {
    return this.profiles().length > 0
      ? this.profiles()
      : [
          { id: 'adult', name: 'Adulto', kind: 'adult', schedule: {}, dns_policy: { safe_search: false }, alert_policy: {} },
          { id: 'child', name: 'Crianca', kind: 'child', schedule: {}, dns_policy: { safe_search: true, blocked_categories: ['adult', 'gambling'] }, alert_policy: {} },
          { id: 'iot', name: 'IoT', kind: 'iot', schedule: {}, dns_policy: { safe_search: false, blocked_categories: ['newly_registered_domains'] }, alert_policy: {} },
          { id: 'guest', name: 'Visitante', kind: 'guest', schedule: {}, dns_policy: { safe_search: false, blocked_categories: ['adult'] }, alert_policy: {} }
        ];
  }

  protected openDeviceSettings(device: Device): void {
    this.selectedDeviceId.set(device.id);
  }

  protected closeDeviceSettings(): void {
    this.selectedDeviceId.set(null);
  }

  protected saveDeviceSettings(device: Device): void {
    this.saveDeviceName(device);
    this.saveProfile(device);
    this.saveDNSPolicy(device);
    this.closeDeviceSettings();
  }

  protected devicePrimaryName(device: Device): string {
    return device.display_name || device.hostname || device.id;
  }

  protected deviceTechnicalName(device: Device): string {
    return device.hostname || device.id;
  }

  protected deviceHasCustomName(device: Device): boolean {
    return !!device.display_name;
  }

  protected alertEvidenceSummary(alert: Alert): string {
    const domain = this.asString(alert.evidence['domain']);
    const category = this.asString(alert.evidence['category']);
    const resolver = this.asString(alert.evidence['resolver']);
    const clientIp = this.asString(alert.evidence['client_ip']);
    const dstIp = this.asString(alert.evidence['dst_ip']);
    const dstPort = this.asString(alert.evidence['dst_port']);

    if (domain) {
      return [domain, category, resolver].filter(Boolean).join(' · ');
    }
    if (dstIp || dstPort) {
      return [dstIp, dstPort ? `porta ${dstPort}` : '', this.asString(alert.evidence['protocol'])].filter(Boolean).join(' · ');
    }
    if (clientIp) {
      return clientIp;
    }

    return 'Sem evidencia resumida';
  }

  protected alertSeverityClass(alert: Alert): string {
    switch (alert.severity) {
      case 'critical':
        return 'critical';
      case 'high':
        return 'high';
      case 'medium':
        return 'medium';
      default:
        return 'low';
    }
  }

  private splitPolicyLines(value: string): string[] {
    return value
      .split(/[\n,]/)
      .map((item) => item.trim().toLowerCase())
      .filter((item, index, items) => item.length > 0 && items.indexOf(item) === index);
  }

  private asString(value: unknown): string {
    return typeof value === 'string' || typeof value === 'number' ? String(value) : '';
  }
}

interface DNSPolicyFormValue {
  blockedDomains: string;
  allowedDomains: string;
  blockedCategories: string;
  safeSearch: boolean;
}
