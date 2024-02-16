package service

import (
	"actlabs-hub/internal/config"
	"actlabs-hub/internal/entity"
	"actlabs-hub/internal/helper"
	"context"
	"fmt"
	"time"

	"golang.org/x/exp/slog"
)

type AutoDestroyService struct {
	appConfig        *config.Config
	serverRepository entity.ServerRepository
	eventService     entity.EventService
}

func NewAutoDestroyService(appConfig *config.Config, repo entity.ServerRepository, eventService entity.EventService) *AutoDestroyService {
	return &AutoDestroyService{
		appConfig:        appConfig,
		serverRepository: repo,
		eventService:     eventService,
	}
}

func (s *AutoDestroyService) MonitorAndDestroyInactiveServers(ctx context.Context) {
	helper.Recoverer(100, "MonitorAndDestroyInactiveServers", func() {
		ticker := time.NewTicker(time.Duration(s.appConfig.ActlabsHubAutoDestroyPollingIntervalSeconds) * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				// Context was cancelled or the application finished, so stop the goroutine
				return
			case <-ticker.C:
				// Every minute, check for servers to destroy
				if err := s.DestroyIdleServers(ctx); err != nil {
					slog.Error("not able to destroy idle servers",
						slog.String("error", err.Error()),
					)
				}
			}
		}
	})
}

func (s *AutoDestroyService) DestroyIdleServers(ctx context.Context) error {
	slog.Info("polling for servers to destroy")
	allServers, err := s.serverRepository.GetAllServersFromDatabase(ctx)
	if err != nil {
		slog.Error("not able to get all servers",
			slog.String("error", err.Error()),
		)
		return err
	}

	for _, server := range allServers {

		lastActivityTime, err := time.Parse(time.RFC3339, server.LastUserActivityTime)
		if err != nil {
			slog.Error("not able to parse last activity time",
				slog.String("error", err.Error()),
			)
			continue
		}

		slog.Debug("checking server for auto destroy",
			slog.String("userPrincipalName", server.UserPrincipalName),
			slog.String("subscriptionId", server.SubscriptionId),
			slog.String("status", string(server.Status)),
			slog.String("lastActivityTime", server.LastUserActivityTime),
			slog.Bool("autoDestroy", server.AutoDestroy),
			slog.Int("inactivityDurationInSeconds", server.InactivityDurationInSeconds),
			slog.Duration("timeSinceLastActivity", time.Since(lastActivityTime)),
			slog.Duration("inactiveDuration", time.Duration(server.InactivityDurationInSeconds)*time.Second),
		)

		if server.Status == entity.ServerStatusRunning && time.Since(lastActivityTime) > time.Duration(server.InactivityDurationInSeconds)*time.Second {
			slog.Debug("server is idle",
				slog.String("userPrincipalName", server.UserPrincipalName),
				slog.String("subscriptionId", server.SubscriptionId),
				slog.String("status", string(server.Status)),
			)
		}

		if server.AutoDestroy &&
			server.Status != entity.ServerStatusAutoDestroyed &&
			server.Status != entity.ServerStatusDestroyed &&
			server.Status != entity.ServerStatusUnregistered &&
			server.Status != entity.ServerStatusRegistered &&
			time.Since(lastActivityTime) > time.Duration(server.InactivityDurationInSeconds)*time.Second &&
			s.VerifyServerIdle(server) {

			slog.Info("destroying server",
				slog.String("userPrincipalName", server.UserPrincipalName),
				slog.String("subscriptionId", server.SubscriptionId),
				slog.String("status", string(server.Status)),
				slog.Bool("autoDestroy", server.AutoDestroy),
				slog.String("lastActivityTime", server.LastUserActivityTime),
				slog.Duration("timeSinceLastActivity", time.Since(lastActivityTime)),
				slog.Duration("inactiveDuration", time.Duration(server.InactivityDurationInSeconds)*time.Second),
			)

			if err := s.DestroyServer(server); err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *AutoDestroyService) VerifyServerIdle(server entity.Server) bool {
	isIdle, err := s.serverRepository.EnsureServerIdle(server)
	if err != nil {
		slog.Error("not able to verify server idle",
			slog.String("userPrincipalName", server.UserPrincipalName),
			slog.String("error", err.Error()),
		)
		return false
	}

	slog.Debug("server idle status",
		slog.String("userPrincipalName", server.UserPrincipalName),
		slog.Bool("isIdle", isIdle),
	)

	return isIdle
}

func (s *AutoDestroyService) DestroyServer(server entity.Server) error {
	if err := s.serverRepository.DestroyServer(server); err != nil {

		slog.Error("not able to destroy server",
			slog.String("userPrincipalName", server.UserPrincipalName),
			slog.String("subscriptionId", server.SubscriptionId),
			slog.String("status", string(server.Status)),
			slog.String("error", err.Error()),
		)

		// Create Event
		s.eventService.CreateEvent(context.Background(), entity.Event{
			Type:      "Normal",
			Reason:    "ServerDestroyedFailed",
			Message:   fmt.Sprintf("Auto Destroyed of server of user %s for subscription %s with version %s failed.", server.UserPrincipalName, server.SubscriptionId, server.Version),
			Reporter:  "actlabs-hub",
			Object:    server.UserPrincipalName,
			TimeStamp: time.Now().Format(time.RFC3339),
		})

		return err
	}

	// update server status in database
	server.Status = entity.ServerStatusAutoDestroyed
	server.DestroyedAtTime = time.Now().Format(time.RFC3339)
	if err := s.serverRepository.UpsertServerInDatabase(server); err != nil {

		slog.Error("not able to destroy server",
			slog.String("userPrincipalName", server.UserPrincipalName),
			slog.String("subscriptionId", server.SubscriptionId),
			slog.String("status", string(server.Status)),
			slog.String("error", err.Error()),
		)

		return fmt.Errorf("server was destroyed but not able to update status in database")
	}

	// Create Event
	s.eventService.CreateEvent(context.Background(), entity.Event{
		Type:      "Normal",
		Reason:    "ServerDestroyed",
		Message:   fmt.Sprintf("Auto Destroyed server of user %s for subscription %s with version %s due to inactivity.", server.UserPrincipalName, server.SubscriptionId, server.Version),
		Reporter:  "actlabs-hub",
		Object:    server.UserPrincipalName,
		TimeStamp: time.Now().Format(time.RFC3339),
	})

	return nil
}
