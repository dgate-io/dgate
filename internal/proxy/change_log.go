package proxy

import (
	"fmt"
	"time"

	"errors"

	"github.com/dgate-io/dgate/pkg/spec"
	"github.com/dgate-io/dgate/pkg/util/sliceutil"
	"github.com/dgraph-io/badger/v4"
	"github.com/mitchellh/mapstructure"
	"github.com/rs/zerolog"
)

// processChangeLog - processes a change log and applies the change to the proxy state
func (ps *ProxyState) processChangeLog(
	cl *spec.ChangeLog, apply, store bool,
) (err error) {
	// TODO: add revert cl function to check if storage fails or something
	if !cl.Cmd.IsNoop() {
		switch cl.Cmd.Resource() {
		case spec.Namespaces:
			var item spec.Namespace
			item, err = decode[spec.Namespace](cl.Item)
			if err == nil {
				ps.logger.Trace().Msgf("Processing namespace: %s", item.Name)
				err = ps.processNamespace(&item, cl)
			}
		case spec.Services:
			var item spec.Service
			item, err = decode[spec.Service](cl.Item)
			if err == nil {
				ps.logger.Trace().Msgf("Processing service: %s", item.Name)
				err = ps.processService(&item, cl)
			}
		case spec.Routes:
			var item spec.Route
			item, err = decode[spec.Route](cl.Item)
			if err == nil {
				ps.logger.Trace().Msgf("Processing route: %s", item.Name)
				err = ps.processRoute(&item, cl)
			}
		case spec.Modules:
			var item spec.Module
			item, err = decode[spec.Module](cl.Item)
			if err == nil {
				ps.logger.Trace().Msgf("Processing module: %s", item.Name)
				err = ps.processModule(&item, cl)
			}
		case spec.Domains:
			var item spec.Domain
			item, err = decode[spec.Domain](cl.Item)
			if err == nil {
				ps.logger.Trace().Msgf("Processing domain: %s", item.Name)
				err = ps.processDomain(&item, cl)
			}
		case spec.Collections:
			var item spec.Collection
			item, err = decode[spec.Collection](cl.Item)
			if err == nil {
				ps.logger.Trace().Msgf("Processing domain: %s", item.Name)
				err = ps.processCollection(&item, cl)
			}
		case spec.Documents:
			var item spec.Document
			item, err = decode[spec.Document](cl.Item)
			if err == nil {
				ps.logger.Trace().Msgf("Processing domain: %s", item.ID)
				err = ps.processDocument(&item, cl)
			}
		default:
			err = fmt.Errorf("unknown command: %s", cl.Cmd)
		}
		if err != nil {
			ps.logger.Err(err).Msg("error processing change log")
			return
		}
	}
	if apply && (cl.Cmd.Resource().IsRelatedTo(spec.Routes) || cl.Cmd.IsNoop()) {
		ps.logger.Trace().Msgf("Registering change log: %s", cl.Cmd)
		errChan := ps.applyChange(cl)
		select {
		case err = <-errChan:
			break
		case <-time.After(time.Second * 15):
			err = errors.New("timeout applying change log")
		}
		if err != nil {
			ps.logger.Err(err).Msg("Error registering change log")
			return
		}
	}
	if store {
		err = ps.store.StoreChangeLog(cl)
		if err != nil {
			// TODO: Add config option to panic on persistent storage errors
			// TODO: maybe revert change here or add to some in-memory queue for changes?
			ps.logger.Err(err).Msg("Error storing change log")
			return
		}
	}
	changeHash, err := HashAny[*spec.ChangeLog](ps.changeHash, cl)
	if err != nil {
		if !ps.config.Debug {
			return err
		}
		ps.logger.Error().
			Err(err).
			Msg("error updating change log hash")
	} else {
		ps.changeHash = changeHash
	}

	return nil
}

func decode[T any](input any) (T, error) {
	var output T
	cfg := &mapstructure.DecoderConfig{
		Metadata:   nil,
		Result:     &output,
		TagName:    "json",
		DecodeHook: mapstructure.StringToTimeHookFunc(time.RFC3339),
	}
	decoder, _ := mapstructure.NewDecoder(cfg)
	err := decoder.Decode(input)
	if err != nil {
		return output, err
	}
	return output, nil
}

func (ps *ProxyState) processNamespace(ns *spec.Namespace, cl *spec.ChangeLog) error {
	switch cl.Cmd.Action() {
	case spec.Add:
		ps.rm.AddNamespace(ns)
		return nil
	case spec.Delete:
		return ps.rm.RemoveNamespace(ns.Name)
	default:
		return fmt.Errorf("unknown command: %s", cl.Cmd)
	}
}

func (ps *ProxyState) processService(svc *spec.Service, cl *spec.ChangeLog) (err error) {
	switch cl.Cmd.Action() {
	case spec.Add:
		_, err = ps.rm.AddService(svc)
	case spec.Delete:
		_, err = ps.rm.AddService(svc)
	default:
		err = fmt.Errorf("unknown command: %s", cl.Cmd)
	}
	return err
}

func (ps *ProxyState) processRoute(route *spec.Route, cl *spec.ChangeLog) (err error) {
	switch cl.Cmd.Action() {
	case spec.Add:
		_, err = ps.rm.AddRoute(route)
	case spec.Delete:
		_, err = ps.rm.AddRoute(route)
	default:
		err = fmt.Errorf("unknown command: %s", cl.Cmd)
	}
	return err
}

func (ps *ProxyState) processModule(module *spec.Module, cl *spec.ChangeLog) (err error) {
	switch cl.Cmd.Action() {
	case spec.Add:
		_, err = ps.rm.AddModule(module)
	case spec.Delete:
		_, err = ps.rm.AddModule(module)
	default:
		err = fmt.Errorf("unknown command: %s", cl.Cmd)
	}
	return err
}

func (ps *ProxyState) processDomain(domain *spec.Domain, cl *spec.ChangeLog) (err error) {
	switch cl.Cmd.Action() {
	case spec.Add:
		_, err = ps.rm.AddDomain(domain)
	case spec.Delete:
		_, err = ps.rm.AddDomain(domain)
	default:
		err = fmt.Errorf("unknown command: %s", cl.Cmd)
	}
	return err
}

func (ps *ProxyState) processCollection(col *spec.Collection, cl *spec.ChangeLog) (err error) {
	switch cl.Cmd.Action() {
	case spec.Add:
		_, err = ps.rm.AddCollection(col)
	case spec.Delete:
		_, err = ps.rm.AddCollection(col)
	default:
		err = fmt.Errorf("unknown command: %s", cl.Cmd)
	}
	return err
}

func (ps *ProxyState) processDocument(doc *spec.Document, cl *spec.ChangeLog) (err error) {
	switch cl.Cmd.Action() {
	case spec.Add:
		err = ps.store.StoreDocument(doc)
	case spec.Delete:
		err = ps.store.DeleteDocument(doc)
	default:
		err = fmt.Errorf("unknown command: %s", cl.Cmd)
	}
	return err
}

// applyChange - apply a change to the proxy state, returns a channel that will receive an error when the state has been updated
func (ps *ProxyState) applyChange(changeLog *spec.ChangeLog) <-chan error {
	done := make(chan error, 1)
	if changeLog == nil {
		changeLog = &spec.ChangeLog{
			Cmd: spec.NoopCommand,
		}
	}
	changeLog.SetErrorChan(done)
	ps.changeChan <- changeLog
	return done
}

func (ps *ProxyState) rollbackChange(changeLog *spec.ChangeLog) {
	if changeLog == nil {
		return
	}
	ps.changeChan <- changeLog
}

func (ps *ProxyState) restoreFromChangeLogs() error {
	// restore state change logs
	logs, err := ps.store.FetchChangeLogs()
	if err != nil {
		if err == badger.ErrKeyNotFound {
			ps.logger.Debug().Msg("no state change logs found in storage")
		} else {
			return errors.New("failed to get state change logs from storage: " + err.Error())
		}
	} else {
		ps.logger.Info().Msg(fmt.Sprintf("restoring %d state change logs from storage", len(logs)))
		// we might need to sort the change logs by timestamp
		for i, cl := range logs {
			ps.logger.Trace().
				Interface("changeLog: "+cl.Name, cl).Msgf("restoring change log index: %d", i)
			lastIteration := i == len(logs)-1
			err = ps.processChangeLog(cl, lastIteration, false)
			if err != nil {
				if ps.config.Debug {
					ps.logger.Err(err).Msg("error processing change log, ignoring while in debug mode")
				} else {
					return err
				}
			}
		}
		// TODO: change to configurable variable
		if len(logs) > 1 {
			removed, err := ps.compactChangeLogs(logs)
			if err != nil {
				ps.logger.Error().Err(err).Msg("failed to compact state change logs")
				return err
			}
			if removed > 0 {
				ps.logger.Info().Msgf("compacted change logs by removing %d out of %d logs", removed, len(logs))
			}
		}
	}
	return nil
}

func (ps *ProxyState) compactChangeLogs(logs []*spec.ChangeLog) (int, error) {
	removeList := compactChangeLogsRemoveList(&ps.logger, sliceutil.SliceCopy(logs))
	removed, err := ps.store.DeleteChangeLogs(removeList)
	if err != nil {
		return removed, err
	}
	return removed, nil
}

/*
compactChangeLogsRemoveList - compacts a list of change logs by removing redundant logs

TODO: perhaps add flag for compacting change logs on startup (mark as experimental)

compaction rules:
- if an add command is followed by a delete command with matching keys, remove both commands
- if an add command is followed by another add command with matching keys, remove the first add command
*/
func compactChangeLogsRemoveList(logger *zerolog.Logger, logs []*spec.ChangeLog) []*spec.ChangeLog {
	removeList := make([]*spec.ChangeLog, 0)
	iterations := 0
START:
	var prevLog *spec.ChangeLog
	// TODO: this can be extended by separating the logs into namespace groups and then compacting each group
	for i := 0; i < len(logs); i++ {
		iterations++
		curLog := logs[i]
		if prevLog != nil {
			if prevLog.Cmd.IsNoop() {
				removeList = append(removeList, prevLog)
				logs = append(logs[:i-1], logs[i:]...)
				goto START
			}
			commonResource := prevLog.Cmd.Resource() == curLog.Cmd.Resource()
			if prevLog.Cmd.Action() == spec.Add && curLog.Cmd.Action() == spec.Delete && commonResource {
				// Rule 1: if an add command is followed by a delete command with matching keys, remove both commands
				if prevLog.Name == curLog.Name && prevLog.Namespace == curLog.Namespace {
					removeList = append(removeList, prevLog, curLog)
					logs = append(logs[:i-1], logs[i+1:]...)
					goto START
				}
			}

			commonAction := prevLog.Cmd.Action() == curLog.Cmd.Action()
			if prevLog.Cmd.Action() == spec.Add && commonAction && commonResource {
				// Rule 2: if an add command is followed by another add command with matching keys, remove the first add command
				if prevLog.Name == curLog.Name && prevLog.Namespace == curLog.Namespace {
					removeList = append(removeList, prevLog)
					logs = append(logs[:i-1], logs[i:]...)
					goto START
				}
			}
		}
		prevLog = curLog
	}
	if logger != nil {
		logger.Debug().Msgf("compacted change logs in %d iterations", iterations)
	}
	// remove duplicates from list
	removeList = sliceutil.SliceUnique(removeList, func(cl *spec.ChangeLog) string { return cl.ID })
	return removeList
}

// Function to check if there is a delete command between two logs with matching keys
// func hasDeleteBetween(logs []*spec.ChangeLog, start, end *spec.ChangeLog) bool {
// 	startIndex := -1
// 	endIndex := -1

// 	for i, log := range logs {
// 		if log.ID == start.ID {
// 			startIndex = i
// 		}
// 		if log.ID == end.ID {
// 			endIndex = i
// 		}
// 	}

// 	if startIndex == -1 || endIndex == -1 {
// 		return false
// 	}

// 	for i := startIndex + 1; i < endIndex; i++ {
// 		if logs[i].Cmd.IsDeleteCommand() {
// 			return true
// 		}
// 	}
// 	return false
// }
