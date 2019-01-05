(function() {
  'use strict';

  angular.module('MyApp', ['ngMaterial', 'ngMessages', 'btford.socket-io',
    'ngCsv'
  ])

  // .config(function($mdIconProvider) {
  //   $mdIconProvider
  //     .iconSet('social', 'img/icons/sets/social-icons.svg', 24)
  //     .iconSet('device', 'img/icons/sets/device-icons.svg', 24)
  //     .iconSet('communication', 'img/icons/sets/communication-icons.svg', 24)
  //     .defaultIconSet('img/icons/sets/core-icons.svg', 24);
  // })
  .config([
    '$locationProvider',
    '$mdThemingProvider',
    function($locationProvider, $mdThemingProvider) {
      $locationProvider.html5Mode(true);
      // $locationProvider.html5mode({ enabled: true, requireBase: false });

      // Available palettes: red, pink, purple, deep-purple, indigo, blue, light-blue, cyan, teal, green, light-green, lime, yellow, amber, orange, deep-orange, brown, grey, blue-grey

      /*
      var customBlueMap = $mdThemingProvider.extendPalette('light-blue', {
        'contrastDefaultColor': 'light',
        'contrastDarkColors': ['50'],
        '50': 'ffffff'
      });
      $mdThemingProvider.definePalette('customBlue', customBlueMap);
      $mdThemingProvider.theme('default')
        .primaryPalette('customBlue', {
          'default': '500',
          'hue-1': '50'
        })
        .accentPalette('pink');

      // Configure a dark theme with primary foreground yellow
      $mdThemingProvider.theme('docs-dark', 'default')
        .primaryPalette('yellow')
        .dark();
      */

      $mdThemingProvider.theme('searchbar')
        // .primaryPalette('light-blue')
        // .accentPalette('pink')
        // .warnPalette('red')
        // .backgroundPalette('grey')
        .dark();

      // Create the other theme options
      var themes = ThemeService();
      for (var index = 0; index < themes.length; ++index) {
        // console.log(themes[index] + '-theme');
        $mdThemingProvider.theme(themes[index] + '-theme')
          .primaryPalette(themes[index]);
      }

      $mdThemingProvider.alwaysWatchTheme(true);

    }
  ])


  .factory('socket', ['socketFactory', function(socketFactory) {
    var socket = socketFactory();
    socket.forward('error');
    return socket;
  }])

  // Reverse display of items
  .filter('reverse', function() {
    return function(items) {
      return items.slice().reverse();
    };
  })


  .controller('DialogController', ['$scope', '$mdDialog',
    function($scope, $mdDialog) {
      $scope.hide = function() {
        $mdDialog.hide();
      };
      $scope.cancel = function() {
        $mdDialog.cancel();
      };
      $scope.answer = function(answer) {
        $mdDialog.hide(answer);
      };
    }
  ])

  .controller('ListCtrl', ['$scope', 'socket', '$location', '$mdDialog',
    '$mdBottomSheet', '$mdSidenav', 'storage',
    function($scope, socket, $location, $mdDialog, $mdBottomSheet,
      $mdSidenav, storage) {

      $scope.theme = storage.get("theme") || 'default';
      $scope.themeList = ThemeService();

      $scope.loading = false;
      $scope.selectedMessage = null;

      // We keep new messages at the start of the array and pop old ones off
      $scope.messages = [];

      $scope.search = {};

      $scope.hashtaginput = storage.get("hashtag");
      $scope.hashtag = null;

      $scope.phone = '';
      $scope.prettyPhone = '';

      // Pause new messages from showing up
      $scope.pause = false;
      $scope.pauseQue = [];

      $scope.poll = storage.get('poll');
      $scope.pollActive = false;

      $scope.showSearch = false;

      // Inform the user
      socket.on("disconnect", function() {
        console.log("client disconnected from server");
        $scope.$apply(function() {
          $scope.hashtag = '';
          $scope.pause = false;
          $scope.pullActive = false;
          $scope.showSearch = false;
        });

        $mdDialog.show(
          $mdDialog.alert()
          // .parent(angular.element(document.querySelector('#popupContainer')))
          .clickOutsideToClose(true)
          .title('Connection Lost')
          // .textContent('')
          .ariaLabel('Connection Lost Alert')
          .ok('Ok')
        );

      });

      // On unpause we need to add the new messages from the que
      $scope.$watch('pause', function(value) {
        if (!value && $scope.pauseQue.length > 0) {
          console.log('adding', $scope.pauseQue.length,
            'new messages');
          $scope.messages.unshift.apply($scope.messages, $scope.pauseQue);
          $scope.pauseQue = [];
          storage.set($scope.hashtag + ':messages', $scope.messages);
        }
      });

      // When the user changes themes we need to act
      $scope.$watch('selectedTheme', function(value) {
        if (value != undefined) {
          $scope.theme = value + '-theme';
          storage.set("theme", $scope.theme);
          console.log('Changed theme', $scope.theme, value);
        }
      });

      $scope.registerHashtag = function($event) {

        $scope.hashtaginput = $scope.hashtaginput.replace('#', '').toLowerCase();

        if ($scope.hashtaginput.length < 2 || $scope.hashtaginput.length >
          20) {
          return;
        }
        console.log('registerHashtag', $scope.hashtaginput, $event);

        socket.emit('register hashtag', $scope.hashtaginput, function(
          data) {
          if (data == 1) {
            $scope.hashtag = $scope.hashtaginput;
            storage.set("hashtag", $scope.hashtaginput);
            $scope.messages = storage.get($scope.hashtag +
              ':messages') || [];
          } else {
            // console.log('ACK from server wtih data: ', data);

            $mdDialog.show(
              $mdDialog.alert()
              // .parent(angular.element(document.querySelector('#popupContainer')))
              .clickOutsideToClose(true)
              .title('Hashtag Taken')
              .textContent('Please choose a different hashtag.')
              .ariaLabel('Hashtag Taken')
              .ok('Ok')
              .targetEvent($event)
            );

          }
        });
      };

      socket.forward('message', $scope);
      $scope.$on('socket:message', function(ev, data) {

        if (!$scope.hashtag) {
          console.log('no hashtag set yet');
          return;
        }

        if ($scope.messages.length > 150) {
          $scope.messages.pop();
        }

        var msg = JSON.parse(data);
        msg.checked = false;

        if ($scope.pause) {
          $scope.pauseQue.unshift(msg);
          // $scope.pauseQue.push(msg);
          console.log('pauseQue contains', $scope.pauseQue.length,
            'messages');
        } else {
          $scope.messages.unshift(msg);
          storage.set($scope.hashtag + ':messages', $scope.messages);
        }

      });

      socket.forward('vote', $scope);
      $scope.$on('socket:vote', function(ev, data) {
        console.log('vote', data);
      });

      // $scope.sendMessage = function (message) {
      //     socket.emit('send_message', message);
      //     // $scope.messages.push($scope.newMessage);
      //     // $scope.newMessage = '';
      // };

      // $scope.idSelectedMessage = null;
      // $scope.setSelected = function (message) {
      //    $scope.idSelectedMessage = $scope.messages.indexOf(message);
      // };

      $scope.selectedMessage = null;
      $scope.setSelectedMessage = function(message) {

        // second click removes it
        if ($scope.selectedMessage == message) {
          $scope.selectedMessage = null;
          if ($scope.poll) {
            $scope.startPoll();
          } else {
            socket.emit('publish', '');
          }

          return;
        }

        $scope.selectedMessage = message;
        //  $scope.messages[$scope.messages.indexOf(message)].active = true

        // $scope.pollActive = false;
        $scope.stopPoll();

        //socket.emit('publish', JSON.stringify({ message: JSON.stringify(message), hashtag: $scope.hashtag }));

        // Hashtag is already known by server
        socket.emit('publish', JSON.stringify({
          type: "message",
          message: message
        }));
      };

      $scope.delete = function(message) {
        console.log("deleting", message);
        $scope.messages.splice($scope.messages.indexOf(message), 1);
      }

      $scope.exportMessages = function() {
        var messages = [];
        for (var key in $scope.messages) {
          var msg = $scope.messages[key];
          messages.push([msg.from, msg.message]);
        }
        return messages;
      }


      $scope.phone = window.phone;
      $scope.prettyPhone = formatPhone($scope.phone);
      // var url = $location.path().split("/")[2]
      // $scope.phone = url;
      // // $scope.$apply(function() { $scope.prettyPhone = formatPhone(url); });
      // $scope.prettyPhone = formatPhone(url);
      // console.log("url path", url)
      // socket.emit("join room", url)
      // window.$loc = $location;


      // We need to delete the property when when unchecking the element
      // so it does not filter the results hiding "checked" messages
      $scope.toggleSearchChecked = function($event) {

        // console.log("clicked", $scope.search);

        if ($scope.search.checked) {
          delete $scope.search.checked;
        } else {
          $scope.search.checked = true;
        }

        // console.log('new search.checked value', $scope.search);
      };



      $scope.deleteMessage = function($event, message) {
        // Appending dialog to document.body to cover sidenav in docs app
        var confirm = $mdDialog.confirm()
          .title('Delete Message?')
          // .textContent('')
          .ariaLabel('Block number')
          .targetEvent($event)
          .ok('Delete')
          .cancel('cancel');
        $mdDialog.show(confirm).then(function() {
          $scope.delete(message);
        }, function() {
          console.log('keep message', message);
        });

        // Next level is "showText"
        $event.stopPropagation();
      };

      // $scope.showText = function(ev, message) {
      //   // Appending dialog to document.body to cover sidenav in docs app
      //   var confirm = $mdDialog.alert()
      //         .clickOutsideToClose(true)
      //         .textContent(message.message)
      //         .ariaLabel(message.message)
      //         .targetEvent(ev)
      //         .ok('close');
      //
      //   $scope.selectedText = message;
      // };


      $scope.toggleSidenav = function(menuId) {
        $mdSidenav(menuId).toggle();
      };


      $scope.showPoll = function(ev) {
        $mdDialog.show({
          controller: 'DialogController',
          clickOutsideToClose: true,
          scope: $scope, // use parent scope in template
          preserveScope: true, // do not forget this if use parent scope
          template: '<md-dialog aria-label="Create Poll" flex> <md-toolbar><div class="md-toolbar-tools"><h2>Create a Poll</h2><span flex></span><md-button class="md-icon-button" ng-click="cancel()"><md-icon class="material-icons">close</md-icon></md-button></div></md-toolbar> <md-dialog-content class="md-padding"> <md-input-container layout="column"><label>Poll Question</label> <input ng-model="poll.question" md-maxlength="200"> </md-input-container> <md-input-container layout="column"> <label>Option A</label> <input ng-model="poll.a" md-maxlength="100"> </md-input-container> <md-input-container layout="column"> <label>Option B</label> <input ng-model="poll.b" md-maxlength="100"> </md-input-container> <md-input-container layout="column"> <label>Option C</label> <input ng-model="poll.c" md-maxlength="100"> </md-input-container> <md-input-container layout="column"> <label>Option D</label> <input ng-model="poll.d" md-maxlength="100"> </md-input-container> </md-dialog-content> <md-dialog-actions> <span flex></span> <md-button ng-click="cancel()"> Close </md-button> <md-button ng-click="clearPoll($event)" class="md-primary"> Clear </md-button> <md-button ng-click="createPoll($event)" class="md-primary"> Publish </md-button> </md-dialog-actions></md-dialog>',
          targetEvent: ev,
        });
        // .then(function(answer) {
        //   $scope.alert = 'You said the information was "' + answer + '".';
        // }, function() {
        //   $scope.alert = 'You cancelled the dialog.';
        // });
      };
      $scope.cancel = function() {
        console.log('$mdDialog.cancel();');
        $mdDialog.cancel();
      };
      $scope.clearPoll = function() {
        $scope.pollActive = false;
        $scope.poll = null;
        storage.set('poll', null);
      }
      $scope.createPoll = function(answer) {
        $scope.pollActive = true;
        storage.set('poll', $scope.poll);
        $mdDialog.hide(answer);
        // socket.emit('poll', $scope.poll);
        console.log('publish poll', $scope.poll);
        socket.emit('publish', JSON.stringify({
          type: "poll",
          message: $scope.poll
        }));
      };
      $scope.stopPoll = function() {
        $scope.pollActive = false;
        // socket.emit('poll', '');
        console.log('stop poll', $scope.poll);

        if (!$scope.selectedMessage) {
          socket.emit('publish', '');
        } else {
          socket.emit('publish', JSON.stringify({
            type: "message",
            message: $scope.selectedMessage
          }));
        }
      }


    }
  ])

  .factory('storage', ['$window', '$log', function($window, $log) {
    var localStorage = $window.localStorage;

    var storage = {
      storage_id: 'LS_', // You can make this whatever you want
      get: function(key) {
        var data, result;

        try {
          data = localStorage.getItem(this.storage_id + key);
        } catch (e) {}

        try {
          result = JSON.parse(data);
        } catch (e) {
          result = data;
        }

        //$log.info('>> storageService',key,result);
        return result;
      },
      set: function(key, data) {
        if (typeof data == "object") {
          data = JSON.stringify(data);
        }

        try {
          localStorage.setItem(this.storage_id + key, data);
        } catch (e) {
          $log.error('!! storageService', e);
        }
      },
      remove: function(key) {
        try {
          var status = localStorage.removeItem(this.storage_id +
            key);
          $log.info('-- storageService', key);
          return status;
        } catch (e) {
          $log.error('!! storageService', e);
          return false;
        }
      },
      clear: function() {
        try {
          localStorage.clear();
          return true;
        } catch (er) {
          return false;
        }
      }
    };

    return storage;

  }]);


  // module.factory('localStorage', localStorage);



  // .controller('ListBottomSheetCtrl', function($scope, $mdBottomSheet) {
  //   $scope.items = [
  //     { name: 'Share', icon: 'share' },
  //     { name: 'Upload', icon: 'upload' },
  //     { name: 'Copy', icon: 'copy' },
  //     { name: 'Print this page', icon: 'print' },
  //   ];
  //
  //   $scope.listItemClick = function($index) {
  //     var clickedItem = $scope.items[$index];
  //     $mdBottomSheet.hide(clickedItem);
  //   };
  // });

  function ThemeService() {
    var themes = [
      'red',
      'pink',
      'purple',
      'deep-purple',
      'indigo',
      'blue',
      'light-blue',
      'cyan',
      'teal',
      'green',
      'light-green',
      'lime',
      'yellow',
      'amber',
      'orange',
      'deep-orange',
      'brown',
      'grey',
      'blue-grey'
    ];

    return themes;
  }

  function formatPhone(phone) {
    return phone.substr(0, 3) + '-' + phone.substr(3, 3) + '-' + phone.substr(
      6, 4);
  }

})(window);
