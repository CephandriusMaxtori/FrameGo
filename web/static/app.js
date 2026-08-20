/* FrameGo admin controller (Alpine.js component, no build step). */
window.framego = function () {
  return {
    display: { width: 800, height: 480, margin: 16, gap: 8, fps: 1, background: "#0b0f14" },
    admin: { enabled: false, bind: "0.0.0.0:8080", token: "" },
    modules: [],
    zones: [],
    moduleTypes: [],
    schemas: {},
    status: [],
    expanded: null,
    newType: "",
    previewT: 0,
    previewError: false,
    toastMsg: "",
    toastType: "",
    toastTimer: null,
    saving: false,
    loading: true,
    dirty: false,
    dragOverZone: null,
    expandTarget: null,

    async init() {
      this.loading = true;
      const [cfg, zones, schemas] = await Promise.all([
        this.api("/api/config"),
        this.api("/api/zones"),
        this.api("/api/schemas"),
      ]);
      if (cfg) {
        Object.assign(this.display, cfg.display);
        Object.assign(this.admin, cfg.admin);
        this.modules = (cfg.modules || []).map((m) => this.decorate(m));
      }
      if (zones) {
        this.zones = zones.zones || [];
        this.moduleTypes = zones.moduleTypes || [];
      }
      if (schemas) {
        this.schemas = schemas;
        for (const [name, s] of Object.entries(schemas)) {
          this.moduleDescriptions[name] = s.description || "";
        }
      }
      this.newType = this.moduleTypes.find((t) => !this.modules.some((m) => m.name === t)) || "";
      this.loading = false;
      this.refreshStatus();
      this.refreshZones();
      setInterval(() => this.refreshPreview(), 2500);
      setInterval(() => this.refreshStatus(), 5000);
    },

    decorate(m) {
      m.options = m.options || {};
      m.optionsErr = "";
      return m;
    },

    async api(url, opts) {
      const headers = { "Content-Type": "application/json" };
      if (this.admin.token) headers["X-FrameGo-Token"] = this.admin.token;
      try {
        const res = await fetch(url, Object.assign({ headers }, opts || {}));
        if (res.status === 401) {
          this.toast("Unauthorized — check the admin token", "error");
          return null;
        }
        if (!res.ok) {
          let msg = "request failed";
          try { msg = (await res.json()).error || msg; } catch (_) {}
          this.toast(msg, "error");
          return null;
        }
        const ct = res.headers.get("content-type") || "";
        return ct.includes("json") ? res.json() : res;
      } catch (err) {
        this.toast("network error: " + err.message, "error");
        return null;
      }
    },

    toast(msg, type) {
      this.toastMsg = msg;
      this.toastType = type || "ok";
      clearTimeout(this.toastTimer);
      this.toastTimer = setTimeout(() => (this.toastMsg = ""), 4000);
    },

    zoneById(id) { return this.zones.find((z) => z.id === id); },
    zoneLabel(id) { const z = this.zoneById(id); return z ? z.label : id; },
    modulesInZone(zoneId) { return this.modules.filter((m) => m.zone === zoneId); },
    isBar(id) { return String(id).endsWith("-bar"); },

    posStyle(z) {
      const w = this.display.width || 800;
      const h = this.display.height || 480;
      return (
        "left:" + ((z.x / w) * 100).toFixed(3) + "%;" +
        "top:" + ((z.y / h) * 100).toFixed(3) + "%;" +
        "width:" + ((z.w / w) * 100).toFixed(3) + "%;" +
        "height:" + ((z.h / h) * 100).toFixed(3) + "%"
      );
    },

    onDragStart(e, name) {
      e.dataTransfer.setData("text/plain", name);
      e.dataTransfer.effectAllowed = "move";
    },
    onDrop(e, zoneId) {
      e.preventDefault();
      this.dragOverZone = null;
      const name = e.dataTransfer.getData("text/plain");
      if (name) this.assignZone(name, zoneId);
    },

    assignZone(name, zoneId) {
      const m = this.modules.find((x) => x.name === name);
      if (m && m.zone !== zoneId) {
        m.zone = zoneId;
        this.dirty = true;
      }
    },

    moveModule(name, dir) {
      const idx = this.modules.findIndex((m) => m.name === name);
      if (idx < 0) return;
      const zone = this.modules[idx].zone;
      const same = this.modules.map((m, i) => ({ m, i })).filter((o) => o.m.zone === zone);
      const pos = same.findIndex((o) => o.m.name === name);
      const target = pos + dir;
      if (target < 0 || target >= same.length) return;
      const a = idx;
      const b = same[target].i;
      [this.modules[a], this.modules[b]] = [this.modules[b], this.modules[a]];
      this.dirty = true;
    },

    defaultZones: {
      clock: "middle-center",
      date: "middle-center",
      weather: "upper-right",
      calendar: "upper-left",
      system: "lower-left",
      moon: "top-right",
      quote: "lower-center",
      nfl: "middle-right",
      smarthome: "lower-right",
      slideshow: "middle-center",
    },

    moduleDescriptions: {},

    addModule(zone) {
      const type = this.newType;
      if (!type) return;
      if (this.modules.some((m) => m.name === type)) {
        this.toast('Module "' + type + '" is already added', "error");
        return;
      }
      const defaultZone = zone || this.defaultZones[type] || "top-left";
      const m = this.decorate({ name: type, zone: defaultZone, visible: true });
      const schema = this.schemas[type];
      if (schema) {
        for (const f of schema.fields) {
          if (f.default) m.options[f.key] = this.parseDefault(f);
        }
      }
      this.modules.push(m);
      this.expanded = type;
      this.newType = this.moduleTypes.find((t) => !this.modules.some((m) => m.name === t)) || "";
      this.dirty = true;
      this.toast("Added " + type + " to " + defaultZone, "ok");
    },

    addModuleToZone(zoneId) {
      const available = this.moduleTypes.filter((t) => !this.modules.some((m) => m.name === t));
      if (available.length === 0) {
        this.toast("All modules already added", "error");
        return;
      }
      this.newType = available[0];
      this.addModule(zoneId);
    },

    parseDefault(f) {
      if (f.kind === "boolean") return f.default === "true";
      if (f.kind === "number") return Number(f.default);
      return f.default;
    },

    removeModule(name) {
      this.modules = this.modules.filter((m) => m.name !== name);
      if (this.expanded === name) this.expanded = null;
      this.dirty = true;
    },

    toggleEdit(name) {
      this.expanded = this.expanded === name ? null : name;
    },

    setField(m, key, val) {
      m.options[key] = val;
      this.dirty = true;
    },

    setFieldNum(m, key, val) {
      const n = Number(val);
      if (!isNaN(n)) this.setField(m, key, n);
    },

    setFieldBool(m, key, val) {
      this.setField(m, key, !!val);
    },

    async save() {
      this.saving = true;
      const cfg = {
        display: {
          width: Number(this.display.width),
          height: Number(this.display.height),
          margin: Number(this.display.margin),
          gap: Number(this.display.gap),
          fps: Number(this.display.fps),
          background: this.display.background,
        },
        admin: { ...this.admin },
        modules: this.modules.map((m) => ({
          name: m.name,
          zone: m.zone,
          visible: m.visible,
          options: m.options || {},
        })),
      };
      const res = await this.api("/api/config", { method: "PUT", body: JSON.stringify(cfg) });
      this.saving = false;
      if (res && res.ok) {
        this.dirty = false;
        this.toast("Saved & reloaded");
        this.refreshZones();
        this.refreshStatus();
        this.refreshPreview();
      }
    },

    async reloadFromDisk() {
      const res = await this.api("/api/reload", { method: "POST" });
      if (res && res.ok) {
        this.toast("Reloaded from disk");
        const cfg = await this.api("/api/config");
        if (cfg) {
          Object.assign(this.display, cfg.display);
          Object.assign(this.admin, cfg.admin);
          this.modules = (cfg.modules || []).map((m) => this.decorate(m));
        }
        this.refreshZones();
        this.refreshStatus();
        this.refreshPreview();
      }
    },

    async refreshZones() {
      const zones = await this.api("/api/zones");
      if (zones) {
        this.zones = zones.zones || [];
        this.moduleTypes = zones.moduleTypes || [];
        this.newType = this.moduleTypes.find((t) => !this.modules.some((m) => m.name === t)) || "";
      }
    },

    async refreshStatus() {
      const st = await this.api("/api/status");
      if (st) this.status = st;
    },

    refreshPreview() {
      this.previewT++;
      this.previewError = false;
    },

    previewSrc() {
      return "/api/snapshot?t=" + this.previewT;
    },

    statusOk() {
      return this.status.some((s) => s.state === "active");
    },
    statusCount() { return this.status.length; },
  };
};
