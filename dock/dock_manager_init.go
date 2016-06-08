/**
 * Copyright (C) 2014 Deepin Technology Co., Ltd.
 *
 * This program is free software; you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation; either version 3 of the License, or
 * (at your option) any later version.
 **/

package dock

import (
	"gir/gio-2.0"
	"path/filepath"
	"pkg.deepin.io/lib/dbus"
	"pkg.deepin.io/lib/dbus/property"
	"time"
)

func (m *DockManager) initDockProperty() error {
	m.dockProperty = NewDockProperty(m)
	err := dbus.InstallOnSession(m.dockProperty)
	return err
}

func (m *DockManager) loadCache() error {
	var err error
	m.desktopWindowsMapCacheManager, err = newDesktopWindowsMapCacheManager(filepath.Join(cacheDir, "desktopWindowsMapCache.gob"))
	if err != nil {
		return err
	}
	m.desktopHashFileMapCacheManager, err = newDesktopHashFileMapCacheManager(filepath.Join(cacheDir, "desktopHashFileMapCache.gob"))
	if err != nil {
		return err
	}
	return nil
}

func (m *DockManager) initEntries() {
	// init entries
	m.desktopWindowsMapCacheManager.SetAutoSaveEnabled(false)
	m.desktopHashFileMapCacheManager.SetAutoSaveEnabled(false)

	m.initDockedApps()
	m.initClientList()

	m.desktopWindowsMapCacheManager.SetAutoSaveEnabled(true)
	m.desktopWindowsMapCacheManager.AutoSave()
	m.desktopHashFileMapCacheManager.SetAutoSaveEnabled(true)
	m.desktopHashFileMapCacheManager.AutoSave()
}

func (m *DockManager) connectSettingKeyChanged(key string, handler func(*gio.Settings, string)) {
	m.settings.Connect("changed::"+key, handler)
}

func (m *DockManager) listenSettingsChanged() {
	// listen hide mode change
	m.connectSettingKeyChanged(settingKeyHideMode, func(g *gio.Settings, key string) {
		mode := HideModeType(g.GetEnum(key))
		logger.Debug(key, "changed to", mode)
		m.updateHideStateWithoutDelay()
	})

	// listen display mode change
	m.connectSettingKeyChanged(settingKeyDisplayMode, func(g *gio.Settings, key string) {
		mode := DisplayModeType(g.GetEnum(key))
		logger.Debug(key, "changed to", mode)

		m.dockHeight = getDockHeightByDisplayMode(mode)
		m.updateDockRect()
	})

	// listen position change
	m.connectSettingKeyChanged(settingKeyPosition, func(g *gio.Settings, key string) {
		position := positionType(g.GetEnum(key))
		logger.Debug(key, "changed to", position)
	})
}

func (m *DockManager) init() error {
	var err error

	m.settings = gio.NewSettings(dockSchema)

	m.HideMode = property.NewGSettingsEnumProperty(m, "HideMode", m.settings, settingKeyHideMode)
	m.DisplayMode = property.NewGSettingsEnumProperty(m, "DisplayMode", m.settings, settingKeyDisplayMode)
	m.Position = property.NewGSettingsEnumProperty(m, "Position", m.settings, settingKeyPosition)

	// ensure init display after init setting
	err = m.initDisplay()
	if err != nil {
		return err
	}
	logger.Info("initialize display done")

	m.FrontendWindow = newPropertyFrontendWindow(m)
	m.FrontendWindow.ConnectChanged(func() {
		logger.Debug("FrontendWindow changed", m.FrontendWindow.Get())
		m.updateHideStateWithoutDelay()
	})

	m.HideState = newPropertyHideState(m)
	m.HideState.ConnectChanged(func() {
		logger.Debug("HideState changed", m.HideState.Get())
	})

	m.smartHideModeTimer = time.AfterFunc(10*time.Second, m.smartHideModeTimerExpired)
	m.smartHideModeTimer.Stop()

	err = m.initDockProperty()
	if err != nil {
		return err
	}
	logger.Info("initialize dock property done")

	m.listenSettingsChanged()

	m.appIdFilterGroup = NewAppIdFilterGroup()
	err = m.loadCache()
	if err != nil {
		return err
	}
	m.dockedAppManager = NewDockedAppManager(m)
	m.initEntries()
	err = dbus.InstallOnSession(m.dockedAppManager)
	if err != nil {
		return err
	}

	err = dbus.InstallOnSession(m)
	if err != nil {
		return err
	}
	dbus.Emit(m, "ServiceRestart")
	return nil
}
